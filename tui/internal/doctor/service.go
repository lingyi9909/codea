package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	runtimedomain "codea/tui/internal/runtime"
	"codea/tui/internal/update"
)

type Status string

const (
	Pass Status = "PASS"
	Warn Status = "WARN"
	Fail Status = "FAIL"
	Skip Status = "SKIP"
)

type Category string

const (
	CategoryStatic Category = "static"
	CategoryConnection Category = "connection"
	CategoryBehavior Category = "behavior"
	CategoryNetwork Category = "network"
)

type Result struct {
	Name string `json:"name"`
	Category Category `json:"category"`
	Status Status `json:"status"`
	Detail string `json:"detail,omitempty"`
	Duration time.Duration `json:"-"`
}

type Report struct {
	StartedAt time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	Results []Result `json:"results"`
}
func (r Report) HasFailures() bool { for _, x := range r.Results { if x.Status == Fail { return true } }; return false }
func (r Report) ExitCode() int { if r.HasFailures() { return 1 }; return 0 }

type Check interface { Run(context.Context) Result }
type checkFunc struct { name string; category Category; fn func(context.Context) (Status, string) }
func (c checkFunc) Run(ctx context.Context) Result { start := time.Now(); status, detail := c.fn(ctx); return Result{Name:c.name, Category:c.category, Status:status, Detail:detail, Duration:time.Since(start)} }

type Service struct { checks []Check; now func() time.Time }
func NewService(checks ...Check) *Service { return &Service{checks:checks, now:time.Now} }
func (s *Service) Run(ctx context.Context) Report { return s.RunWithProgress(ctx, nil, nil) }

// RunWithProgress preserves the normal Doctor report contract while allowing
// interactive callers to surface which check is currently running. Callbacks
// are synchronous and optional; non-interactive callers can keep using Run.
func (s *Service) RunWithProgress(ctx context.Context, onStart func(string, Category), onResult func(Result)) Report {
	now:=s.now; if now==nil { now=time.Now }; r:=Report{StartedAt:now().UTC()}
	for _, check := range s.checks {
		select { case <-ctx.Done():
			res:=Result{Name:"Doctor 上下文",Category:CategoryBehavior,Status:Fail,Detail:ctx.Err().Error()}
			r.Results=append(r.Results,res); if onResult!=nil{onResult(res)}; r.FinishedAt=now().UTC(); return r
		default: }
		if onStart!=nil {
			if c,ok:=check.(checkFunc);ok { onStart(c.name,c.category) }
		}
		res:=check.Run(ctx)
		r.Results=append(r.Results,res)
		if onResult!=nil { onResult(res) }
	}
	r.FinishedAt=now().UTC(); return r
}

type Config struct {
	HomeDir string
	ConfigDir string
	Runtime runtimedomain.AgentRuntime
	RuntimeStartError error
	RuntimeURL string
	ExpectedOpenCodeVersion string
	BehaviorTimeout time.Duration
}

func NewDefaultService(cfg Config) (*Service,error) {
	if strings.TrimSpace(cfg.HomeDir)=="" { return nil,fmt.Errorf("home dir is required") }
	home,err:=filepath.Abs(cfg.HomeDir); if err!=nil{return nil,err}
	configDir:=cfg.ConfigDir; if configDir==""{configDir=filepath.Join(home,"runtime-config")}; if !filepath.IsAbs(configDir){configDir=filepath.Join(home,configDir)}
	expected:=strings.TrimSpace(cfg.ExpectedOpenCodeVersion); if expected==""{expected="1.18.11"}
	timeout:=cfg.BehaviorTimeout; if timeout<=0{timeout=20*time.Second}
	checks:=defaultChecks(defaultCheckConfig{home:home,configDir:configDir,runtime:cfg.Runtime,runtimeStartErr:cfg.RuntimeStartError,runtimeURL:cfg.RuntimeURL,expectedVersion:expected,behaviorTimeout:timeout})
	return NewService(checks...),nil
}

// NewCandidateService runs the same Doctor contract against a staged/installed
// update candidate instead of resolving the currently active version pointer.
// This is the bridge Task 18 uses for both pre-switch V2+C2-temp and post-switch
// validation.
func NewCandidateService(candidate update.Candidate, rt runtimedomain.AgentRuntime, runtimeURL, expectedOpenCodeVersion string, timeout time.Duration) (*Service,error) {
	if strings.TrimSpace(candidate.VersionDir)==""||strings.TrimSpace(candidate.ConfigDir)==""{return nil,fmt.Errorf("candidate version/config dir required")}
	versionDir,err:=filepath.Abs(candidate.VersionDir);if err!=nil{return nil,err}
	configDir,err:=filepath.Abs(candidate.ConfigDir);if err!=nil{return nil,err}
	home:=filepath.Dir(filepath.Dir(versionDir))
	expected:=strings.TrimSpace(expectedOpenCodeVersion);if expected==""{expected="1.18.11"}
	if timeout<=0{timeout=20*time.Second}
	checks:=defaultChecks(defaultCheckConfig{home:home,configDir:configDir,releaseRoot:versionDir,runtime:rt,runtimeURL:runtimeURL,expectedVersion:expected,behaviorTimeout:timeout})
	return NewService(checks...),nil
}

func Init(homeDir,configDir string) error {
	if strings.TrimSpace(homeDir)=="" { return fmt.Errorf("home dir is required") }
	home,err:=filepath.Abs(homeDir); if err!=nil{return err}
	if configDir==""{configDir=filepath.Join(home,"runtime-config")}; if !filepath.IsAbs(configDir){configDir=filepath.Join(home,configDir)}
	dirs:=[]string{
		filepath.Join(home,"versions"), filepath.Join(home,"staging"), filepath.Join(home,"backups"), filepath.Join(home,"transactions"), filepath.Join(home,"bin"),
		filepath.Join(configDir,"codea"), filepath.Join(configDir,"skills"), filepath.Join(configDir,"agents"),
		filepath.Join(configDir,"xdg","config"), filepath.Join(configDir,"xdg","data"), filepath.Join(configDir,"xdg","cache"), filepath.Join(configDir,"xdg","state"),
	}
	for _,dir:=range dirs { if err:=os.MkdirAll(dir,0o700);err!=nil{return fmt.Errorf("create %s: %w",dir,err)} }
	p:=filepath.Join(configDir,"codea","config.json")
	f,err:=os.OpenFile(p,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0o600)
	if errors.Is(err,os.ErrExist){return nil}; if err!=nil{return err}
	if _,err:=f.WriteString("{\n  \"schemaVersion\": 1\n}\n");err!=nil{f.Close();_ = os.Remove(p);return err}
	if err:=f.Sync();err!=nil{f.Close();_ = os.Remove(p);return err}; return f.Close()
}
