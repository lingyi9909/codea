package update

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type fakeChecker struct { failPhase CheckPhase; calls []CheckPhase }
func (f *fakeChecker) Check(_ context.Context, phase CheckPhase, _ Candidate) error { f.calls = append(f.calls, phase); if phase == f.failPhase { return errors.New("doctor failed") }; return nil }

func writeFile(t *testing.T, path, body string, mode os.FileMode) { t.Helper(); if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { t.Fatal(err) }; if err := os.WriteFile(path, []byte(body), mode); err != nil { t.Fatal(err) } }
func buildRelease(t *testing.T, root, version string, configSchema int) string {
	t.Helper(); writeFile(t, filepath.Join(root, "VERSION"), version+"\n", 0o644)
	codea, opencode := "codea", "opencode"; if runtime.GOOS == "windows" { codea += ".exe"; opencode += ".exe" }
	writeFile(t, filepath.Join(root, "bin", codea), "codea", 0o755); writeFile(t, filepath.Join(root, "bin", opencode), "opencode", 0o755)
	writeFile(t, filepath.Join(root, "plugins", "index.js"), "export default {};\n", 0o644)
	writeFile(t, filepath.Join(root, "agents", "api-documentation", "agent.md"), "agent", 0o644)
	writeFile(t, filepath.Join(root, "skills", "api-documentation", "SKILL.md"), "skill", 0o644)
	if configSchema > 0 { writeFile(t, filepath.Join(root, "config", "codea-schema-version"), string(rune('0'+configSchema))+"\n", 0o644) }
	generateManifest(t, root); return root
}
func generateManifest(t *testing.T, root string) {
	t.Helper(); type entry struct { Path string `json:"path"`; SHA256 string `json:"sha256"`; Size int64 `json:"size"` }; var files []entry
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error { if err != nil { return err }; if d.IsDir() || d.Name()=="manifest.json" { return nil }; rel,_:=filepath.Rel(root,path); rel=filepath.ToSlash(rel); data,_:=os.ReadFile(path); h:=sha256.Sum256(data); st,_:=os.Stat(path); files=append(files,entry{rel,hex.EncodeToString(h[:]),st.Size()}); return nil }); if err != nil { t.Fatal(err) }
	m:=map[string]any{"schemaVersion":1,"algorithm":"sha256","files":files}; b,_:=json.MarshalIndent(m,"","  "); writeFile(t,filepath.Join(root,"manifest.json"),string(b)+"\n",0o644)
}
func makeHomeWithV1(t *testing.T) string { t.Helper(); home:=t.TempDir(); v1:=filepath.Join(home,"versions","1.0.0"); buildRelease(t,v1,"1.0.0",1); sw:=NewPlatformSwitcher(home); if err:=sw.Switch(v1); err!=nil { t.Fatal(err) }; writeFile(t,filepath.Join(home,"runtime-config","codea","config.json"),`{"schemaVersion":1,"name":"keep"}`+"\n",0o600); return home }
func newService(t *testing.T, home string, checker CandidateChecker, migrations *MigrationRegistry) *UpdateService { t.Helper(); s,err:=NewService(ServiceConfig{HomeDir:home,Checker:checker,Migrations:migrations}); if err!=nil { t.Fatal(err) }; return s }

func TestVerifyReleaseRejectsTamperAndUnmanifested(t *testing.T) {
	root:=buildRelease(t,filepath.Join(t.TempDir(),"release"),"2.0.0",1); v:=Verifier{}; if _,err:=v.Verify(root); err!=nil { t.Fatalf("valid release rejected: %v",err) }
	writeFile(t,filepath.Join(root,"plugins","index.js"),"export default [];\n",0o644); if _,err:=v.Verify(root); err==nil || !strings.Contains(err.Error(),"checksum") { t.Fatalf("tamper not rejected: %v",err) }
	buildRelease(t,root,"2.0.0",1); writeFile(t,filepath.Join(root,"extra.txt"),"extra",0o644); if _,err:=v.Verify(root); err==nil || !strings.Contains(err.Error(),"unmanifested") { t.Fatalf("extra not rejected: %v",err) }
}
func TestMigrationRegistryRequiresEveryHopAndDoesNotMutateSource(t *testing.T) {
	r:=NewMigrationRegistry(); r.Register(1,func(in map[string]any)(map[string]any,error){in["schemaVersion"]=2;in["added"]="yes";return in,nil}); src:=map[string]any{"schemaVersion":1,"nested":map[string]any{"x":"y"}}
	out,err:=r.Migrate(src,1,2); if err!=nil { t.Fatal(err) }; if out["added"]!="yes" || out["schemaVersion"].(int)!=2 { t.Fatalf("bad migration: %#v",out) }; if _,ok:=src["added"];ok { t.Fatal("source mutated") }; if _,err:=r.Migrate(src,1,3); err==nil || !strings.Contains(err.Error(),"missing migration") { t.Fatalf("missing hop not rejected: %v",err) }
}
func TestUpgradeCommitsVersionAndMigratedConfig(t *testing.T) {
	home:=makeHomeWithV1(t); release:=buildRelease(t,filepath.Join(t.TempDir(),"v2"),"2.0.0",2); reg:=NewMigrationRegistry(); reg.Register(1,func(in map[string]any)(map[string]any,error){in["schemaVersion"]=2;in["migrated"]=true;return in,nil}); checker:=&fakeChecker{}; s:=newService(t,home,checker,reg)
	if err:=s.Upgrade(context.Background(),release);err!=nil{t.Fatalf("upgrade: %v",err)}; current,err:=s.switcher.Current();if err!=nil{t.Fatal(err)};if filepath.Base(current)!="2.0.0"{t.Fatalf("current=%s",current)};b,_:=os.ReadFile(filepath.Join(home,"runtime-config","codea","config.json"));if !strings.Contains(string(b),`"migrated": true`){t.Fatalf("config not migrated: %s",b)};tx,err:=s.journal.Load();if err!=nil{t.Fatal(err)};if tx.Status!=TxCommitted{t.Fatalf("status=%s",tx.Status)};if len(checker.calls)!=2||checker.calls[0]!=CheckPreSwitch||checker.calls[1]!=CheckPostSwitch{t.Fatalf("checker calls=%v",checker.calls)}
}
func TestUpgradePostCheckFailureRollsBackVersionAndConfig(t *testing.T) {
	home:=makeHomeWithV1(t);before,_:=os.ReadFile(filepath.Join(home,"runtime-config","codea","config.json"));release:=buildRelease(t,filepath.Join(t.TempDir(),"v2"),"2.0.0",2);reg:=NewMigrationRegistry();reg.Register(1,func(in map[string]any)(map[string]any,error){in["schemaVersion"]=2;return in,nil});s:=newService(t,home,&fakeChecker{failPhase:CheckPostSwitch},reg);if err:=s.Upgrade(context.Background(),release);err==nil{t.Fatal("expected failure")};current,_:=s.switcher.Current();if filepath.Base(current)!="1.0.0"{t.Fatalf("not rolled back: %s",current)};after,_:=os.ReadFile(filepath.Join(home,"runtime-config","codea","config.json"));if string(after)!=string(before){t.Fatalf("config changed after failure: %s",after)};if _,err:=os.Stat(filepath.Join(home,"versions","2.0.0"));!os.IsNotExist(err){t.Fatalf("failed target left installed: %v",err)};tx,_:=s.journal.Load();if tx.Status!=TxRolledBack{t.Fatalf("tx status=%s",tx.Status)}
}
func TestRecoverPendingRollsBackCrashAfterSwitch(t *testing.T) {
	home:=makeHomeWithV1(t);v2:=buildRelease(t,filepath.Join(home,"versions","2.0.0"),"2.0.0",1);sw:=NewPlatformSwitcher(home);if err:=sw.Switch(v2);err!=nil{t.Fatal(err)};j:=NewJournal(home);tx:=&Transaction{ID:"crash",FromVersion:"1.0.0",ToVersion:"2.0.0",Status:TxPending,ConfigBackupPath:filepath.Join(home,"backups","crash","runtime-config")};tx.MarkStep("install-version",TxCommitted,"");tx.MarkStep("switch-current",TxCommitted,"");if err:=j.Save(tx);err!=nil{t.Fatal(err)};s:=newService(t,home,&fakeChecker{},NewMigrationRegistry());if err:=s.Recover(context.Background());err!=nil{t.Fatal(err)};current,_:=s.switcher.Current();if filepath.Base(current)!="1.0.0"{t.Fatalf("recover current=%s",current)};tx,_=j.Load();if tx.Status!=TxRolledBack{t.Fatalf("status=%s",tx.Status)}
}
func TestRollbackLastCommittedRestoresConfigAndVersion(t *testing.T) {
	home:=makeHomeWithV1(t);release:=buildRelease(t,filepath.Join(t.TempDir(),"v2"),"2.0.0",1);s:=newService(t,home,&fakeChecker{},NewMigrationRegistry());if err:=s.Upgrade(context.Background(),release);err!=nil{t.Fatal(err)};writeFile(t,filepath.Join(home,"runtime-config","codea","config.json"),`{"schemaVersion":1,"name":"changed-after-upgrade"}`+"\n",0o600);if err:=s.Rollback(context.Background());err!=nil{t.Fatal(err)};current,_:=s.switcher.Current();if filepath.Base(current)!="1.0.0"{t.Fatalf("rollback current=%s",current)};b,_:=os.ReadFile(filepath.Join(home,"runtime-config","codea","config.json"));if !strings.Contains(string(b),`"name":"keep"`){t.Fatalf("backup not restored: %s",b)}
}
func TestPreparePackageRejectsZipTraversal(t *testing.T) {
	p:=filepath.Join(t.TempDir(),"bad.zip");f,err:=os.Create(p);if err!=nil{t.Fatal(err)};zw:=zip.NewWriter(f);w,_:=zw.Create("../escape");_,_=w.Write([]byte("x"));_=zw.Close();_=f.Close();_,err=PreparePackage(p,filepath.Join(t.TempDir(),"stage"));if err==nil||!strings.Contains(err.Error(),"unsafe"){t.Fatalf("zip traversal not rejected: %v",err)}
}
func TestMigrationFailureLeavesCurrentUntouchedAndNoJournal(t *testing.T) {
	home:=makeHomeWithV1(t);before,_:=os.ReadFile(filepath.Join(home,"runtime-config","codea","config.json"));release:=buildRelease(t,filepath.Join(t.TempDir(),"v2"),"2.0.0",2);reg:=NewMigrationRegistry();reg.Register(1,func(in map[string]any)(map[string]any,error){return nil,errors.New("migration boom")});s:=newService(t,home,&fakeChecker{},reg);if err:=s.Upgrade(context.Background(),release);err==nil||!strings.Contains(err.Error(),"migration"){t.Fatalf("expected migration failure: %v",err)};after,_:=os.ReadFile(filepath.Join(home,"runtime-config","codea","config.json"));if string(after)!=string(before){t.Fatal("current config was modified")};if _,err:=os.Stat(filepath.Join(home,"update_journal.json"));!os.IsNotExist(err){t.Fatalf("journal should not begin before prechecks: %v",err)}
}
func TestRecoverPendingSwitchIntentObservesFilesystem(t *testing.T) {
	home:=makeHomeWithV1(t);v2:=buildRelease(t,filepath.Join(home,"versions","2.0.0"),"2.0.0",1);sw:=NewPlatformSwitcher(home);if err:=sw.Switch(v2);err!=nil{t.Fatal(err)};j:=NewJournal(home);tx:=&Transaction{ID:"crash-pending",FromVersion:"1.0.0",ToVersion:"2.0.0",Status:TxPending};tx.MarkStep("install-version",TxCommitted,"");tx.MarkStep("switch-current",TxPending,"");if err:=j.Save(tx);err!=nil{t.Fatal(err)};s:=newService(t,home,&fakeChecker{},NewMigrationRegistry());if err:=s.Recover(context.Background());err!=nil{t.Fatal(err)};current,_:=sw.Current();if filepath.Base(current)!="1.0.0"{t.Fatalf("pending switch was not recovered: %s",current)}
}
func TestUpgradeRejectsDowngrade(t *testing.T) {
	home:=makeHomeWithV1(t);_=os.RemoveAll(filepath.Join(home,"versions","1.0.0"));v2:=buildRelease(t,filepath.Join(home,"versions","2.0.0"),"2.0.0",1);if err:=NewPlatformSwitcher(home).Switch(v2);err!=nil{t.Fatal(err)};release:=buildRelease(t,filepath.Join(t.TempDir(),"v1"),"1.5.0",1);s:=newService(t,home,&fakeChecker{},NewMigrationRegistry());if err:=s.Upgrade(context.Background(),release);err==nil||!strings.Contains(err.Error(),"newer"){t.Fatalf("downgrade not rejected: %v",err)}
}
func TestUpdateLockRejectsConcurrentUpgrade(t *testing.T) {
	home:=makeHomeWithV1(t);lock,err:=acquireUpdateLock(home);if err!=nil{t.Fatal(err)};defer lock.Release();release:=buildRelease(t,filepath.Join(t.TempDir(),"v2"),"2.0.0",1);s:=newService(t,home,&fakeChecker{},NewMigrationRegistry());if err:=s.Upgrade(context.Background(),release);err==nil||!strings.Contains(err.Error(),"another update"){t.Fatalf("concurrent update not rejected: %v",err)}
}
func TestBasicCheckerRejectsIncompleteCandidate(t *testing.T) {
	root:=buildRelease(t,filepath.Join(t.TempDir(),"release"),"2.0.0",1);cfg:=filepath.Join(t.TempDir(),"cfg");if err:=os.MkdirAll(cfg,0o700);err!=nil{t.Fatal(err)};checker:=BasicChecker{};if err:=checker.Check(context.Background(),CheckPreSwitch,Candidate{Version:"2.0.0",VersionDir:root,ConfigDir:cfg});err!=nil{t.Fatalf("valid candidate rejected: %v",err)};codea:="codea";if runtime.GOOS=="windows"{codea+=".exe"};if err:=os.Remove(filepath.Join(root,"bin",codea));err!=nil{t.Fatal(err)};if err:=checker.Check(context.Background(),CheckPreSwitch,Candidate{Version:"2.0.0",VersionDir:root,ConfigDir:cfg});err==nil{t.Fatal("missing codea binary was not rejected")}
}
func TestFreshInstallVersionSwitch(t *testing.T) {
	home:=t.TempDir();v1:=buildRelease(t,filepath.Join(home,"versions","1.0.0"),"1.0.0",1);sw:=NewPlatformSwitcher(home);if err:=sw.Switch(v1);err!=nil{t.Fatal(err)};current,err:=sw.Current();if err!=nil{t.Fatal(err)};if current!=v1{t.Fatalf("current=%s want=%s",current,v1)}
}
