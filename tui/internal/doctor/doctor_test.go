package doctor

import (
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
	"time"

	runtimedomain "codea/tui/internal/runtime"
	"codea/tui/internal/update"
)

type fakeRuntime struct {
	health runtimedomain.HealthInfo
	healthErr error
	subscribeErr error
	promptErr error
	agents []runtimedomain.Agent
	events chan runtimedomain.Event
}
func (f *fakeRuntime) Health(context.Context)(runtimedomain.HealthInfo,error){return f.health,f.healthErr}
func (f *fakeRuntime) CreateSession(context.Context,runtimedomain.CreateSessionRequest)(runtimedomain.Session,error){return runtimedomain.Session{ID:"doctor-session"},nil}
func (f *fakeRuntime) Prompt(_ context.Context,id runtimedomain.SessionID,_ runtimedomain.PromptRequest) error {if f.promptErr!=nil{return f.promptErr};if f.events!=nil{f.events<-runtimedomain.Event{SessionID:string(id),Content:"CODEA_DOCTOR_OK"}};return nil}
func (f *fakeRuntime) Subscribe(context.Context)(<-chan runtimedomain.Event,error){if f.subscribeErr!=nil{return nil,f.subscribeErr};if f.events==nil{f.events=make(chan runtimedomain.Event,4)};return f.events,nil}
func (f *fakeRuntime) ReplyApproval(context.Context,runtimedomain.ApprovalID,runtimedomain.ApprovalReply)error{return nil}
func (f *fakeRuntime) Cancel(context.Context,runtimedomain.SessionID)error{return nil}
func (f *fakeRuntime) ListAgents(context.Context)([]runtimedomain.Agent,error){return f.agents,nil}
func (f *fakeRuntime) ListModels(context.Context)([]runtimedomain.Model,error){return nil,nil}
func (f *fakeRuntime) ListSessions(context.Context)([]runtimedomain.Session,error){return nil,nil}
func (f *fakeRuntime) GetSessionMessages(context.Context,runtimedomain.SessionID)([]runtimedomain.Message,error){return nil,nil}
func (f *fakeRuntime) CompactSession(context.Context,runtimedomain.SessionID)error{return nil}
func (f *fakeRuntime) Capabilities()runtimedomain.RuntimeCapabilities{return runtimedomain.RuntimeCapabilities{}}

func doctorWrite(t *testing.T,path,body string){t.Helper();if err:=os.MkdirAll(filepath.Dir(path),0755);err!=nil{t.Fatal(err)};if err:=os.WriteFile(path,[]byte(body),0644);err!=nil{t.Fatal(err)}}
func doctorManifest(t *testing.T,root string){t.Helper();type entry struct{Path string `json:"path"`;SHA256 string `json:"sha256"`;Size int64 `json:"size"`};var files []entry;err:=filepath.WalkDir(root,func(path string,d os.DirEntry,err error)error{if err!=nil{return err};if d.IsDir()||d.Name()=="manifest.json"{return nil};rel,_:=filepath.Rel(root,path);b,_:=os.ReadFile(path);h:=sha256.Sum256(b);st,_:=os.Stat(path);files=append(files,entry{filepath.ToSlash(rel),hex.EncodeToString(h[:]),st.Size()});return nil});if err!=nil{t.Fatal(err)};b,_:=json.MarshalIndent(map[string]any{"schemaVersion":1,"algorithm":"sha256","files":files},"","  ");doctorWrite(t,filepath.Join(root,"manifest.json"),string(b)+"\n")}
func doctorInstall(t *testing.T)(string,string){t.Helper();home:=t.TempDir();version:="1.0.0";root:=filepath.Join(home,"versions",version);doctorWrite(t,filepath.Join(root,"VERSION"),version+"\n");codea,oc:="codea","opencode";if runtime.GOOS=="windows"{codea+=".exe";oc+=".exe"};doctorWrite(t,filepath.Join(root,"bin",codea),"bin");doctorWrite(t,filepath.Join(root,"bin",oc),"bin");doctorWrite(t,filepath.Join(root,"plugins","index.js"),"export default {};\n");doctorWrite(t,filepath.Join(root,"skills","code-review","SKILL.md"),"# skill\n");doctorWrite(t,filepath.Join(root,"agents","code-reviewer","manifest.yaml"),"name: code-reviewer\n");doctorWrite(t,filepath.Join(root,"agents","code-reviewer","agent.md"),"agent\n");doctorWrite(t,filepath.Join(root,"config","opencode","permissions.json"),`{"agents":{"general":{}}}`+"\n");doctorWrite(t,filepath.Join(root,"runtime-version.json"),`{"openCodeVersion":"1.18.11"}`+"\n");doctorManifest(t,root);sw:=update.NewPlatformSwitcher(home);if err:=sw.Switch(root);err!=nil{t.Fatal(err)};cfg:=filepath.Join(home,"runtime-config");doctorWrite(t,filepath.Join(cfg,"codea","config.json"),`{"schemaVersion":1}`+"\n");return home,cfg}
func resultByName(t *testing.T,r Report,name string)Result{t.Helper();for _,x:=range r.Results{if x.Name==name{return x}};t.Fatalf("missing result %s",name);return Result{}}

func TestDefaultDoctorAllChecksPassWithHealthyRuntime(t *testing.T){home,cfg:=doctorInstall(t);rt:=&fakeRuntime{health:runtimedomain.HealthInfo{Healthy:true,Version:"1.18.11"},agents:[]runtimedomain.Agent{{Name:"code-reviewer"},{Name:"unit-test-generator"},{Name:"api-documentation"}}};svc,err:=NewDefaultService(Config{HomeDir:home,ConfigDir:cfg,Runtime:rt,RuntimeURL:"http://127.0.0.1:4141",ExpectedOpenCodeVersion:"1.18.11",BehaviorTimeout:time.Second});if err!=nil{t.Fatal(err)};report:=svc.Run(context.Background());if report.HasFailures(){t.Fatalf("unexpected failures: %s",FormatText(report))};for _,name:=range []string{"发行包完整性","配置 Schema","权限配置","Skill Manifest","Agent Manifest","Plugin Bundle","版本兼容","Runtime 健康","企业 Agent","模型连接","SSE","模型推理","Runtime 网络绑定"}{if got:=resultByName(t,report,name).Status;got!=Pass{t.Fatalf("%s=%s",name,got)}}}
func TestDoctorFailsTamperedReleaseAndUnhealthyRuntime(t *testing.T){home,cfg:=doctorInstall(t);current,_:=update.NewPlatformSwitcher(home).Current();doctorWrite(t,filepath.Join(current,"plugins","index.js"),"tampered\n");rt:=&fakeRuntime{health:runtimedomain.HealthInfo{Healthy:false,Version:"1.18.11"}};svc,_:=NewDefaultService(Config{HomeDir:home,ConfigDir:cfg,Runtime:rt,RuntimeURL:"http://127.0.0.1:4141",ExpectedOpenCodeVersion:"1.18.11",BehaviorTimeout:50*time.Millisecond});report:=svc.Run(context.Background());if !report.HasFailures()||report.ExitCode()!=1{t.Fatalf("report should fail: %s",FormatText(report))};if resultByName(t,report,"发行包完整性").Status!=Fail{t.Fatal("tamper not failed")};if resultByName(t,report,"Runtime 健康").Status!=Fail{t.Fatal("health not failed")}}
func TestDoctorSkipsBehaviorWhenRuntimeStartupFailed(t *testing.T){home,cfg:=doctorInstall(t);var rt *fakeRuntime;svc,_:=NewDefaultService(Config{HomeDir:home,ConfigDir:cfg,Runtime:rt,RuntimeStartError:errors.New("runtime boom"),RuntimeURL:"http://127.0.0.1:4141"});report:=svc.Run(context.Background());if resultByName(t,report,"Runtime 健康").Status!=Fail{t.Fatal("startup failure must fail health")};for _,name:=range []string{"企业 Agent","模型连接","SSE","模型推理"}{if resultByName(t,report,name).Status!=Skip{t.Fatalf("%s must skip when runtime startup failed",name)}}}
func TestDoctorRejectsNonLoopbackRuntimeURL(t *testing.T){home,cfg:=doctorInstall(t);svc,_:=NewDefaultService(Config{HomeDir:home,ConfigDir:cfg,Runtime:&fakeRuntime{health:runtimedomain.HealthInfo{Healthy:true,Version:"1.18.11"}},RuntimeURL:"http://0.0.0.0:4141",ExpectedOpenCodeVersion:"1.18.11"});report:=svc.Run(context.Background());r:=resultByName(t,report,"Runtime 网络绑定");if r.Status!=Fail||!strings.Contains(r.Detail,"loopback"){t.Fatalf("network result=%+v",r)}}
func TestInitCreatesConfigWithoutOverwritingExisting(t *testing.T){home:=t.TempDir();cfg:=filepath.Join(home,"runtime-config");if err:=Init(home,cfg);err!=nil{t.Fatal(err)};p:=filepath.Join(cfg,"codea","config.json");b,err:=os.ReadFile(p);if err!=nil{t.Fatal(err)};if !strings.Contains(string(b),`"schemaVersion": 1`){t.Fatalf("unexpected config %s",b)};doctorWrite(t,p,"CUSTOM\n");if err:=Init(home,cfg);err!=nil{t.Fatal(err)};b,_=os.ReadFile(p);if string(b)!="CUSTOM\n"{t.Fatal("init overwrote existing config")}}
func TestInferenceFailureIsReportedForModelConnectionAndInference(t *testing.T){home,cfg:=doctorInstall(t);rt:=&fakeRuntime{health:runtimedomain.HealthInfo{Healthy:true,Version:"1.18.11"},promptErr:errors.New("model unavailable")};svc,_:=NewDefaultService(Config{HomeDir:home,ConfigDir:cfg,Runtime:rt,RuntimeURL:"http://127.0.0.1:4141",ExpectedOpenCodeVersion:"1.18.11",BehaviorTimeout:time.Second});report:=svc.Run(context.Background());if resultByName(t,report,"模型连接").Status!=Fail||resultByName(t,report,"模型推理").Status!=Fail{t.Fatalf("model probe should fail both results: %s",FormatText(report))}}
