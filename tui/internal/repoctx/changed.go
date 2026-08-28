package repoctx

import (
	"context"
	"os/exec"
	"sort"
	"strings"
)

type CommandRunner interface { Run(context.Context,string,...string)([]byte,error) }
type execRunner struct{}
func (execRunner) Run(ctx context.Context,name string,args ...string)([]byte,error) { return exec.CommandContext(ctx,name,args...).Output() }

func GitChangedFiles(ctx context.Context,root string,runner CommandRunner) []string {
	if runner==nil { runner=execRunner{} }
	out,err:=runner.Run(ctx,"git","-C",root,"status","--porcelain=v1","-z","--untracked-files=all"); if err!=nil{return nil}
	recs:=strings.Split(string(out),"\x00"); set:=map[string]struct{}{}
	for i:=0;i<len(recs);i++ { rec:=recs[i]; if len(rec)<3{continue}; status:=rec[:2]; path:=rec[3:]; if path==""{continue}; if status[0]=='R'||status[0]=='C'{if i+1<len(recs){i++}}; if norm,ok:=normalizeRelativePath(path);ok{set[norm]=struct{}{}}else{p:=normalizeSlash(path);if p!=""&&!strings.HasPrefix(p,"../"){set[p]=struct{}{}}} }
	paths:=make([]string,0,len(set)); for p:=range set{paths=append(paths,p)}; sort.Strings(paths); return paths
}
