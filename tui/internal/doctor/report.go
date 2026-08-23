package doctor

import (
	"fmt"
	"strings"
)

func FormatText(r Report) string {
	var b strings.Builder
	b.WriteString("Codea Doctor\n")
	b.WriteString("============\n")
	order:=[]Category{CategoryStatic,CategoryConnection,CategoryBehavior,CategoryNetwork}
	labels:=map[Category]string{CategoryStatic:"静态检查",CategoryConnection:"连接检查",CategoryBehavior:"行为检查",CategoryNetwork:"网络检查"}
	for _,cat:=range order{
		wrote:=false
		for _,res:=range r.Results{
			if res.Category!=cat{continue}
			if !wrote{fmt.Fprintf(&b,"\n[%s]\n",labels[cat]);wrote=true}
			if res.Detail!=""{fmt.Fprintf(&b,"%-5s %s - %s\n",res.Status,res.Name,res.Detail)}else{fmt.Fprintf(&b,"%-5s %s\n",res.Status,res.Name)}
		}
	}
	pass,warn,fail,skip:=0,0,0,0
	for _,res:=range r.Results{switch res.Status{case Pass:pass++;case Warn:warn++;case Fail:fail++;case Skip:skip++}}
	fmt.Fprintf(&b,"\n汇总：PASS=%d WARN=%d FAIL=%d SKIP=%d\n",pass,warn,fail,skip)
	return b.String()
}
