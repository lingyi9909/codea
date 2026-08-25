# Codea V1 私有大模型配置与使用指南

> 适用版本：Codea `0.1.0` / OpenCode Runtime `v1.18.11`  
> 主要环境：Windows 10/11 x64；同时附 macOS 使用说明  
> 目标读者：第一次下载安装 Codea、需要接入公司或个人私有大模型的用户

---

## 1. 先看结论

Codea 不要求使用公网大模型。

只要你的模型服务提供 **OpenAI Compatible API**，例如能够提供类似下面的接口：

```text
POST http://your-llm-host/v1/chat/completions
```

就可以作为 Codea 的模型后端。

典型场景包括：

- 公司内部统一大模型网关
- vLLM
- SGLang
- LiteLLM
- Ollama 的 OpenAI Compatible 接口
- 其他兼容 OpenAI Chat Completions API 的私有服务

Codea V1 当前没有“模型配置向导”页面。第一次配置私有模型时，需要编辑一次：

```text
%USERPROFILE%\.codea\runtime-config\opencode.json
```

配置完成后，日常使用只需要进入项目目录执行：

```powershell
codea
```

Codea 会自动启动随安装包提供的 OpenCode `v1.18.11` Runtime，不需要另外启动 OpenCode。

---

# 2. Codea、OpenCode 和私有模型之间是什么关系

运行关系可以简单理解成：

```text
你
 ↓
Codea TUI
 ↓
Codea Runtime Contract
 ↓
OpenCode v1.18.11
 ↓
公司私有模型 / 本地模型
```

其中：

- **Codea**：你直接使用的产品界面，负责会话、Agent、Skill、安全策略、工具调用等。
- **OpenCode**：Codea V1 当前使用的 Runtime，由 Codea 自动拉起和管理。
- **私有模型**：真正负责推理和生成内容的 LLM 服务。

因此，“给 Codea 配置自己的模型”，本质上就是给 **Codea 自己管理的 OpenCode Runtime** 配置 Provider 和 Model。

Codea 使用独立配置目录，不会修改你电脑原来已有的 OpenCode 用户配置。

默认目录：

```text
Windows: %USERPROFILE%\.codea\runtime-config
macOS:   ~/.codea/runtime-config
```

模型配置文件：

```text
opencode.json
```

Codea 启动时会在这个配置中注册自己的 Plugin，但会保留你写入的 `model`、`provider` 等配置。

---

# 3. Windows 第一次安装

## 3.1 解压安装包

假设下载的是：

```text
codea-0.1.0-windows-x64.zip
```

解压后目录类似：

```text
codea-0.1.0-windows-x64\
├── bin\
├── plugins\
├── skills\
├── agents\
├── config\
├── install\
│   └── install.ps1
├── VERSION
├── runtime-version.json
└── manifest.json
```

不要只复制 `codea.exe`。安装包里的 OpenCode Runtime、Plugin、Agent、Skill 都属于 Codea V1 的完整运行环境。

## 3.2 执行安装

打开 PowerShell，进入解压后的目录，例如：

```powershell
cd C:\Users\yourname\Downloads\codea-0.1.0-windows-x64
```

如果 PowerShell 当前策略不允许执行脚本，可以只对当前窗口临时放开：

```powershell
Set-ExecutionPolicy -Scope Process Bypass
```

然后执行：

```powershell
.\install\install.ps1 -PackageDir .
```

安装成功后默认目录为：

```text
%USERPROFILE%\.codea
```

其中命令入口为：

```text
%USERPROFILE%\.codea\bin\codea.cmd
```

如果执行 `codea` 提示找不到命令，把下面目录加入当前用户 `PATH`：

```text
%USERPROFILE%\.codea\bin
```

临时在当前 PowerShell 中测试，可以执行：

```powershell
$env:Path = "$env:USERPROFILE\.codea\bin;$env:Path"
```

然后：

```powershell
codea init
codea doctor
```

推荐第一次使用时先跑一遍 `doctor`。

---

# 4. 配置自己的私有大模型

下面是本指南最重要的一部分。

## 4.1 先准备 3 个信息

向你们公司大模型平台负责人确认：

```text
1. Base URL
2. Model ID
3. API Key / Token（如果服务需要）
```

例如：

```text
Base URL: http://10.10.20.30:8000/v1
Model ID: qwen3-coder-480b
API Key:  company-xxxxxx
```

这里的 **Model ID 必须使用服务端真实接受的 model 值**，不是你自己随便起的显示名称。

如果不确定，可以先测试模型服务是否能访问。

PowerShell 示例：

```powershell
$headers = @{
  "Authorization" = "Bearer $env:CODEA_LLM_API_KEY"
  "Content-Type"  = "application/json"
}

$body = @{
  model = "qwen3-coder-480b"
  messages = @(
    @{ role = "user"; content = "回复 OK" }
  )
} | ConvertTo-Json -Depth 5

Invoke-RestMethod `
  -Uri "http://10.10.20.30:8000/v1/chat/completions" `
  -Method Post `
  -Headers $headers `
  -Body $body
```

如果这个请求本身都无法成功，先解决模型服务、网络或鉴权问题，再启动 Codea。

---

## 4.2 初始化 Codea 配置目录

执行：

```powershell
codea init
```

默认会创建/准备：

```text
%USERPROFILE%\.codea
```

模型配置目录为：

```text
%USERPROFILE%\.codea\runtime-config
```

如果 `opencode.json` 不存在，可以手动创建。

完整路径：

```text
%USERPROFILE%\.codea\runtime-config\opencode.json
```

---

## 4.3 推荐配置：OpenAI Compatible 私有模型

假设你的模型信息是：

```text
Provider ID: company
Base URL:    http://10.10.20.30:8000/v1
Model ID:    qwen3-coder-480b
```

把 `opencode.json` 写成：

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "company/qwen3-coder-480b",
  "provider": {
    "company": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "公司私有大模型",
      "options": {
        "baseURL": "http://10.10.20.30:8000/v1",
        "apiKey": "{env:CODEA_LLM_API_KEY}"
      },
      "models": {
        "qwen3-coder-480b": {
          "name": "Qwen3 Coder 480B"
        }
      }
    }
  }
}
```

注意：Codea V1 锁定的是 OpenCode `v1.18.11`，这里使用的是 **OpenCode V1 的 `provider` 配置格式**。不要直接复制未来 OpenCode V2 文档里的 `providers/package/settings` 格式。

### 字段解释

`company`

```json
"provider": {
  "company": {}
}
```

这是 Provider ID，可以自己起名字，但必须和 `model` 前缀一致。

例如：

```json
"model": "company/qwen3-coder-480b"
```

表示：

```text
Provider = company
Model    = qwen3-coder-480b
```

`npm`

```json
"npm": "@ai-sdk/openai-compatible"
```

表示模型服务使用 OpenAI Compatible Chat Completions 协议。

`baseURL`

```json
"baseURL": "http://10.10.20.30:8000/v1"
```

写到 `/v1` 即可，不要把 `/chat/completions` 再拼进去。

`apiKey`

```json
"apiKey": "{env:CODEA_LLM_API_KEY}"
```

推荐通过环境变量读取，不要把真实 Key 直接写进配置文件。

---

# 5. 配置 API Key

## 5.1 当前 PowerShell 临时设置

```powershell
$env:CODEA_LLM_API_KEY = "你的真实Key"
```

只在当前 PowerShell 窗口有效，关闭窗口后失效。

这种方式最适合第一次测试。

## 5.2 Windows 用户环境变量长期保存

确认测试没有问题后，可以执行：

```powershell
[Environment]::SetEnvironmentVariable(
  "CODEA_LLM_API_KEY",
  "你的真实Key",
  "User"
)
```

然后关闭当前终端，重新打开 PowerShell。

确认：

```powershell
$env:CODEA_LLM_API_KEY
```

生产或公司环境中，建议使用公司的凭据管理方案，不要把 Key 提交到 Git，也不要写进项目源码。

---

# 6. 不需要 API Key 的内网模型

如果公司模型接口没有鉴权，可以省略 `apiKey`：

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "company/qwen3-coder-480b",
  "provider": {
    "company": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "公司私有大模型",
      "options": {
        "baseURL": "http://10.10.20.30:8000/v1"
      },
      "models": {
        "qwen3-coder-480b": {
          "name": "Qwen3 Coder 480B"
        }
      }
    }
  }
}
```

---

# 7. 公司网关需要自定义 Header

有的公司网关不用标准 `Authorization: Bearer ...`，而是例如：

```text
X-API-Key: xxxxxx
X-Tenant: codea
```

可以使用：

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "company/qwen3-coder-480b",
  "provider": {
    "company": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "公司大模型网关",
      "options": {
        "baseURL": "http://llm-gateway.company.internal/v1",
        "headers": {
          "X-API-Key": "{env:COMPANY_LLM_KEY}",
          "X-Tenant": "codea"
        }
      },
      "models": {
        "qwen3-coder-480b": {
          "name": "Qwen3 Coder 480B"
        }
      }
    }
  }
}
```

然后设置：

```powershell
$env:COMPANY_LLM_KEY = "你的Key"
```

---

# 8. 本机 Ollama 示例

如果只是个人电脑快速体验，也可以连接 Ollama 的 OpenAI Compatible 接口。

假设已经在 Ollama 中准备好模型：

```text
qwen3-coder
```

配置：

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "ollama/qwen3-coder",
  "provider": {
    "ollama": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "Ollama Local",
      "options": {
        "baseURL": "http://127.0.0.1:11434/v1"
      },
      "models": {
        "qwen3-coder": {
          "name": "Qwen3 Coder"
        }
      }
    }
  }
}
```

本地小模型的 Agent/Tool 能力通常明显弱于大型 Coder 模型。Codea 能否稳定完成复杂 Review、单测生成、Bug 修复，最终仍取决于模型自身的代码能力、上下文长度和 Tool Calling 能力。

---

# 9. 配完后怎么验证

建议按下面顺序验证，不要直接上来就测试复杂项目。

## 第一步：检查配置 JSON

PowerShell：

```powershell
Get-Content "$env:USERPROFILE\.codea\runtime-config\opencode.json"
```

确认：

- JSON 没有多余逗号
- Provider ID 正确
- Model ID 正确
- Base URL 正确
- Key 没有直接硬编码（推荐）

## 第二步：运行 Doctor

```powershell
codea doctor
```

Doctor 会检查 Codea 本地环境及 Runtime 状态。

如果这里出现 FAIL，先不要开始实际项目任务。

## 第三步：在一个测试项目中启动 Codea

例如：

```powershell
cd D:\workspace\demo-project
codea
```

Codea 会以**当前目录作为项目工作目录**。

不要在 `C:\Users\...` 这类大目录直接启动，否则 Agent 的项目边界会过大，也不利于权限控制。

## 第四步：先发一个只读问题

例如：

```text
先不要修改代码。分析一下这个项目是什么技术栈，主要模块有哪些。
```

如果能够正常返回结果，说明：

```text
Codea → OpenCode Runtime → 私有模型
```

这条主链路已经通了。

---

# 10. 日常怎么启动

以后使用非常简单。

进入你要开发的项目根目录：

```powershell
cd D:\workspace\your-project
```

启动：

```powershell
codea
```

正常情况下不需要手工执行：

```text
opencode serve
```

Codea 会自动启动和停止自己安装包中锁定的 OpenCode Runtime。

---

# 11. Codea 界面怎么用

当前 V1 是 TUI（终端界面）。

底部快捷键：

| 快捷键 | 功能 |
|---|---|
| `Enter` | 提交当前输入 |
| `Alt + Enter` | 输入换行 |
| `Ctrl + T` | 展开 / 收起思考过程 |
| `Ctrl + S` | 打开 / 关闭历史 Session |
| `Ctrl + K` | 打开 / 关闭 Skills 页面 |
| `Ctrl + L` | 清理当前聊天显示 |
| `Ctrl + C` | 退出 Codea |

如果终端太小，Codea 会要求至少：

```text
70 x 20
```

---

# 12. V1 到底应该怎么“发任务”

当前 Codea V1 的产品入口采用 **General Agent**。

也就是说，你不需要先手工选择：

```text
Code Reviewer
Unit Test Generator
API Documentation
General
```

日常更推荐直接用自然语言说明任务和边界。

例如：

## 12.1 Code Review

```text
Review 当前工作区的代码变更。
重点检查：
1. 是否存在功能 Bug；
2. 是否有空指针、并发、事务或数据一致性问题；
3. 是否引入安全风险；
4. 每个问题给出文件和代码位置；
5. 先只 Review，不修改代码。
```

## 12.2 生成单元测试

```text
分析当前这次代码变更，为受影响逻辑补充单元测试。
先给 Test Plan，再生成测试代码。
不要修改生产代码。
完成后执行相关测试，并告诉我通过/失败情况。
```

## 12.3 分析并修复 Bug

```text
这个接口现在出现 xxx 问题。
请先分析调用链和可能原因，找到根因后再修改。
修改完成后补充或更新测试，并执行验证。
```

## 12.4 生成接口文档

```text
分析当前 Controller 及相关 DTO，生成 API 文档。
请求字段、响应字段和错误码只能依据代码事实，不要猜测不存在的字段。
```

## 12.5 普通研发任务

```text
先理解这个需求和现有实现，然后完成开发。
修改范围只限于与本需求直接相关的代码。
开发完成后执行相应测试，并总结修改内容和验证结果。
```

---

# 13. Tool Approval 怎么选

当 Agent 要执行需要授权的操作时，Codea 会弹出权限确认。

V1 的决定只有三种：

```text
Allow Once   仅本次允许
Always       本次规则范围持续允许
Reject       拒绝
```

建议第一次试用时：

- 读代码、跑安全测试：确认命令后可以 `Allow Once`
- 写文件：确认路径和目的后再允许
- 不认识的命令：`Reject`
- 删除文件、危险 Shell、访问敏感路径：默认拒绝

不要因为“是 AI 发起的”就无条件 Always。

---

# 14. Skills 怎么使用

按：

```text
Ctrl + K
```

进入 Skills 页面。

Codea 会显示可见的 Skill，并依据当前 Skill Mode 决定哪些 Skill 可以加载。

V1 支持三种策略思想：

- 企业受控能力
- General Strict
- General Compatible

公司正式环境建议以受控 / Strict 为主；个人试用可以根据需要使用 Compatible。

Skill Mode 也可以通过环境变量设置，例如：

```powershell
$env:CODEA_SKILL_MODE = "strict"
```

如果公司要求只允许审批过的 Skills，还可以使用：

```powershell
$env:CODEA_APPROVED_SKILLS = "skill-a,skill-b"
```

如果没有特殊要求，第一次试用建议先保持默认，不要同时修改多个策略变量。

---

# 15. Session 怎么使用

Codea 会维护 Runtime Session。

按：

```text
Ctrl + S
```

可以打开历史 Session。

使用方向键选择，`Enter` 恢复。

同一个开发问题尽量在同一个 Session 中继续，不需要每次重新把背景说一遍。

如果已经切换到完全不同的任务，建议新开一次 Codea / 新 Session，避免上下文互相污染。

---

# 16. 思考过程和工具执行怎么看

Codea 会把：

```text
最终回答
Reasoning / Thinking
Tool 执行状态
```

分开显示。

`Ctrl + T` 可以展开或折叠 Thinking。

工具执行过程中会看到类似：

```text
◌ tool-name
✓ tool-name
✗ tool-name
```

分别代表：

```text
执行中
成功
失败
```

判断任务是否真的完成时，不要只看模型最后一句话，也要看相关 Tool / Test 是否真实执行成功。

---

# 17. 更换模型怎么做

如果一个 Provider 下配置了多个模型，例如：

```json
{
  "model": "company/qwen3-coder-480b",
  "provider": {
    "company": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "公司模型",
      "options": {
        "baseURL": "http://llm.company.internal/v1",
        "apiKey": "{env:CODEA_LLM_API_KEY}"
      },
      "models": {
        "qwen3-coder-480b": {
          "name": "Qwen3 Coder 480B"
        },
        "deepseek-v3.1": {
          "name": "DeepSeek V3.1"
        }
      }
    }
  }
}
```

默认模型由最上面的：

```json
"model": "company/qwen3-coder-480b"
```

决定。

想切换时修改为：

```json
"model": "company/deepseek-v3.1"
```

然后退出并重新启动 Codea。

V1 暂时建议使用这种明确的配置方式，避免不同 Session 中模型不一致导致结果难以复现。

---

# 18. 推荐的私有模型能力要求

Codea 是 Coding Agent，不是普通聊天机器人。

仅仅“能回答文本”并不代表适合 Codea。

推荐私有模型至少具备：

1. 较强代码理解能力
2. Tool Calling / Function Calling 能力
3. 较长上下文
4. 稳定的结构化输出能力
5. 能连续多轮调用工具
6. 对 Java / Go / TypeScript / Shell 等研发内容有较强能力

如果模型经常出现下面问题：

```text
不调用工具
工具参数 JSON 经常错误
读了代码仍然编造字段
复杂任务执行几步后忘记目标
上下文稍长就明显退化
```

大概率是模型能力问题，而不是 Codea 配置问题。

第一次公司试点建议优先选择当前私有模型平台中 **Coding / Agent 能力最强** 的模型，不建议为了节省资源先拿小参数模型作为验收基线。

---

# 19. 常见问题排查

## 19.1 `codea` 找不到

检查：

```powershell
Test-Path "$env:USERPROFILE\.codea\bin\codea.cmd"
```

如果返回：

```text
True
```

说明已安装，只是 PATH 没配置。

临时加入：

```powershell
$env:Path = "$env:USERPROFILE\.codea\bin;$env:Path"
```

---

## 19.2 `codea doctor` Runtime FAIL

优先检查：

```text
1. 安装包是否完整
2. opencode.json 是否是合法 JSON
3. 私有模型 Base URL 是否可达
4. API Key 是否存在
5. Model ID 是否正确
```

特别注意：Codea 对损坏的 `opencode.json` 是 fail-closed，不会悄悄覆盖错误配置。

---

## 19.3 401 / 403

通常是鉴权问题。

确认：

```powershell
$env:CODEA_LLM_API_KEY
```

再确认公司网关到底需要：

```text
Authorization: Bearer xxx
```

还是：

```text
X-API-Key: xxx
```

---

## 19.4 404

重点检查 Base URL。

推荐：

```text
http://host:port/v1
```

而不是：

```text
http://host:port/v1/chat/completions
```

OpenAI Compatible Provider 会自己拼接具体 API 路径。

---

## 19.5 `model not found`

配置里的 Model ID 和服务端真实 Model ID 不一致。

例如服务端需要：

```text
Qwen3-Coder-480B-A35B-Instruct
```

你不能随意写成：

```text
qwen-coder
```

除非网关本身做了模型别名映射。

---

## 19.6 能聊天，但是不会调用工具

主链路已经通了，但模型 Agent 能力不足或模型 API 没正确支持 Tool Calling。

先用简单任务验证：

```text
读取当前项目的 go.mod 或 pom.xml，并告诉我项目依赖。
```

如果模型始终只“解释应该怎么读”，却不实际调用工具，优先检查模型的 Tool Calling 支持。

---

## 19.7 请求很慢或容易超时

检查：

- 模型推理速度
- 公司网关排队时间
- 模型最大输出 Token
- 上下文是否过大
- 模型服务是否限流

复杂 Agent 任务通常不是一次 LLM 请求，而是：

```text
模型推理 → Tool → 模型推理 → Tool → ... → 最终回答
```

因此私有模型单次响应延迟会被多轮 Agent 调用放大。

---

## 19.8 HTTPS 内网证书报错

如果公司模型网关使用公司自签 CA，不建议关闭 TLS 校验。

正确方式是把公司 CA 安装到操作系统/运行环境信任链中。

不要为了“先跑起来”长期使用跳过证书校验的方式。

---

# 20. 明天第一次试用建议按这个顺序

不要一上来就用正式大项目做复杂修改。

### Step 1：安装

```powershell
.\install\install.ps1 -PackageDir .
```

### Step 2：加入 PATH

```powershell
$env:Path = "$env:USERPROFILE\.codea\bin;$env:Path"
```

### Step 3：初始化

```powershell
codea init
```

### Step 4：配置私有模型

创建：

```text
%USERPROFILE%\.codea\runtime-config\opencode.json
```

### Step 5：设置 Key

```powershell
$env:CODEA_LLM_API_KEY = "xxxx"
```

### Step 6：Doctor

```powershell
codea doctor
```

### Step 7：进入测试项目

```powershell
cd D:\workspace\demo-project
codea
```

### Step 8：只读测试

输入：

```text
不要修改代码，先分析这个项目的技术栈和主要模块。
```

### Step 9：Code Review 测试

输入：

```text
Review 当前 Git 代码变更，只报告真实问题，不要修改代码。每个问题给出文件和代码位置。
```

### Step 10：单测测试

输入：

```text
根据当前变更补充单元测试。不要修改生产代码。完成后执行测试并报告结果。
```

完成这 10 步之后，再开始正式试用复杂开发能力。

---

# 21. 推荐的公司私有模型配置模板

实际落地时可以直接复制下面模板，只替换 4 个位置。

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "company/YOUR_MODEL_ID",
  "provider": {
    "company": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "Company Private LLM",
      "options": {
        "baseURL": "http://YOUR_LLM_HOST:PORT/v1",
        "apiKey": "{env:CODEA_LLM_API_KEY}"
      },
      "models": {
        "YOUR_MODEL_ID": {
          "name": "YOUR MODEL DISPLAY NAME"
        }
      }
    }
  }
}
```

需要替换：

```text
YOUR_MODEL_ID
YOUR_LLM_HOST
PORT
YOUR MODEL DISPLAY NAME
```

然后：

```powershell
$env:CODEA_LLM_API_KEY = "你的Key"
codea doctor
codea
```

---

# 22. 当前 V1 使用边界

为了避免第一次试用产生错误预期，明确几个 V1 边界：

1. **目前没有图形化模型配置向导**，私有模型通过 `opencode.json` 配置。
2. **顶层交互入口目前是 General Agent**，用户直接描述目标即可。
3. Codea 会自动管理锁定的 OpenCode Runtime，不要求用户单独安装/启动 OpenCode。
4. Codea 的 Plugin、Agent、Skill 都来自当前安装版本，不建议手工修改安装目录里的文件。
5. 私有模型是否能完成复杂 Coding Agent 任务，与模型本身的 Coding、Tool Calling、上下文和稳定性直接相关。
6. 公司断网环境下，模型服务和企业依赖源必须是内网可访问地址；不要在正式配置中引用公网模型 API。
7. G15 公司内网镜像 Release Certification 当前按项目验收决定保持 deferred，不影响 Codea V1 的本轮试用。

---

# 23. 如果你只想最快跑起来

Windows 用户第一次试用只记住下面几步即可：

```powershell
# 1. 安装
.\install\install.ps1 -PackageDir .

# 2. 临时 PATH
$env:Path = "$env:USERPROFILE\.codea\bin;$env:Path"

# 3. 初始化
codea init

# 4. 设置私有模型 Key
$env:CODEA_LLM_API_KEY = "xxxx"

# 5. 编辑
notepad "$env:USERPROFILE\.codea\runtime-config\opencode.json"

# 6. 检查
codea doctor

# 7. 在项目根目录启动
cd D:\workspace\your-project
codea
```

`opencode.json` 最小模板：

```json
{
  "model": "company/YOUR_MODEL_ID",
  "provider": {
    "company": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "Company Private LLM",
      "options": {
        "baseURL": "http://YOUR_LLM_HOST:PORT/v1",
        "apiKey": "{env:CODEA_LLM_API_KEY}"
      },
      "models": {
        "YOUR_MODEL_ID": {
          "name": "Private Model"
        }
      }
    }
  }
}
```

这套配置完成后，后续每天通常只需要：

```powershell
cd 你的项目根目录
codea
```

即可开始使用。

---

# 附录 A：macOS

macOS 安装包包含：

```text
install/install.sh
```

安装后默认目录同样是：

```text
~/.codea
```

配置文件：

```text
~/.codea/runtime-config/opencode.json
```

环境变量：

```bash
export CODEA_LLM_API_KEY='xxxx'
```

然后：

```bash
codea init
codea doctor
cd ~/workspace/your-project
codea
```

私有模型 JSON 配置与 Windows 相同。

---

# 附录 B：参考

Codea V1 当前锁定 OpenCode Runtime：

```text
v1.18.11
```

OpenCode V1 Provider 配置参考：

```text
https://opencode.ai/docs/providers/
```

请注意 OpenCode 后续 V2 的 Provider Schema 已发生变化。本项目 V1 的配置文档应以锁定 Runtime 的 V1 配置格式和 Codea 自己的回归测试为准。
