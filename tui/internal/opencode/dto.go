// Code generated from OpenAPI spec. DO NOT EDIT.

package opencode

type OpenCodeAPIErrorData struct {
	IsRetryable     bool              `json:"isRetryable"`
	Message         string            `json:"message"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	ResponseBody    string            `json:"responseBody,omitempty"`
	ResponseHeaders map[string]string `json:"responseHeaders,omitempty"`
	StatusCode      int64             `json:"statusCode,omitempty"`
}

type OpenCodeAPIError struct {
	Data OpenCodeAPIErrorData `json:"data"`
	Name string               `json:"name"`
}

type OpenCodeAgentModel struct {
	ModelID    string `json:"modelID"`
	ProviderID string `json:"providerID"`
}

type OpenCodeAgent struct {
	Color       string                    `json:"color,omitempty"`
	Description string                    `json:"description,omitempty"`
	Hidden      bool                      `json:"hidden,omitempty"`
	Mode        string                    `json:"mode"`
	Model       *OpenCodeAgentModel       `json:"model,omitempty"`
	Name        string                    `json:"name"`
	Native      bool                      `json:"native,omitempty"`
	Options     map[string]any            `json:"options"`
	Permission  OpenCodePermissionRuleset `json:"permission"`
	Prompt      string                    `json:"prompt,omitempty"`
	Steps       float64                   `json:"steps,omitempty"`
	Temperature float64                   `json:"temperature,omitempty"`
	TopP        float64                   `json:"topP,omitempty"`
	Variant     string                    `json:"variant,omitempty"`
}

type OpenCodeAgentColor any

type OpenCodeAgentConfig struct {
	Color       any                      `json:"color,omitempty"`
	Description string                   `json:"description,omitempty"`
	Disable     bool                     `json:"disable,omitempty"`
	Hidden      bool                     `json:"hidden,omitempty"`
	MaxSteps    int64                    `json:"maxSteps,omitempty"`
	Mode        string                   `json:"mode,omitempty"`
	Model       string                   `json:"model,omitempty"`
	Options     map[string]any           `json:"options,omitempty"`
	Permission  OpenCodePermissionConfig `json:"permission,omitempty"`
	Prompt      string                   `json:"prompt,omitempty"`
	Steps       int64                    `json:"steps,omitempty"`
	Temperature float64                  `json:"temperature,omitempty"`
	Tools       map[string]bool          `json:"tools,omitempty"`
	TopP        float64                  `json:"top_p,omitempty"`
	Variant     string                   `json:"variant,omitempty"`
}

type OpenCodeAgentPartSource struct {
	End   int64  `json:"end"`
	Start int64  `json:"start"`
	Value string `json:"value"`
}

type OpenCodeAgentPart struct {
	ID        string                   `json:"id"`
	MessageID string                   `json:"messageID"`
	Name      string                   `json:"name"`
	SessionID string                   `json:"sessionID"`
	Source    *OpenCodeAgentPartSource `json:"source,omitempty"`
	Type      string                   `json:"type"`
}

type OpenCodeAgentPartInputSource struct {
	End   int64  `json:"end"`
	Start int64  `json:"start"`
	Value string `json:"value"`
}

type OpenCodeAgentPartInput struct {
	ID     string                        `json:"id,omitempty"`
	Name   string                        `json:"name"`
	Source *OpenCodeAgentPartInputSource `json:"source,omitempty"`
	Type   string                        `json:"type"`
}

type OpenCodeAgentV2Info struct {
	Color       OpenCodeAgentColor          `json:"color,omitempty"`
	Description string                      `json:"description,omitempty"`
	Hidden      bool                        `json:"hidden"`
	ID          string                      `json:"id"`
	Mode        string                      `json:"mode"`
	Model       OpenCodeModelRef            `json:"model,omitempty"`
	Permissions OpenCodePermissionV2Ruleset `json:"permissions"`
	Request     OpenCodeProviderRequest     `json:"request"`
	Steps       int64                       `json:"steps,omitempty"`
	System      string                      `json:"system,omitempty"`
}

type OpenCodeApiAuth struct {
	Key      string            `json:"key"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Type     string            `json:"type"`
}

type OpenCodeAssistantMessagePath struct {
	Cwd  string `json:"cwd"`
	Root string `json:"root"`
}

type OpenCodeAssistantMessageTime struct {
	Completed int64 `json:"completed,omitempty"`
	Created   int64 `json:"created"`
}

type OpenCodeAssistantMessageTokensCache struct {
	Read  float64 `json:"read"`
	Write float64 `json:"write"`
}

type OpenCodeAssistantMessageTokens struct {
	Cache     OpenCodeAssistantMessageTokensCache `json:"cache"`
	Input     float64                             `json:"input"`
	Output    float64                             `json:"output"`
	Reasoning float64                             `json:"reasoning"`
	Total     float64                             `json:"total,omitempty"`
}

type OpenCodeAssistantMessage struct {
	Agent      string                         `json:"agent"`
	Cost       float64                        `json:"cost"`
	Error      any                            `json:"error,omitempty"`
	Finish     string                         `json:"finish,omitempty"`
	ID         string                         `json:"id"`
	Mode       string                         `json:"mode"`
	ModelID    string                         `json:"modelID"`
	ParentID   string                         `json:"parentID"`
	Path       OpenCodeAssistantMessagePath   `json:"path"`
	ProviderID string                         `json:"providerID"`
	Role       string                         `json:"role"`
	SessionID  string                         `json:"sessionID"`
	Structured any                            `json:"structured,omitempty"`
	Summary    bool                           `json:"summary,omitempty"`
	Time       OpenCodeAssistantMessageTime   `json:"time"`
	Tokens     OpenCodeAssistantMessageTokens `json:"tokens"`
	Variant    string                         `json:"variant,omitempty"`
}

type OpenCodeAttachmentConfig struct {
	Image OpenCodeImageAttachmentConfig `json:"image,omitempty"`
}

type OpenCodeAuth any

type OpenCodeBadRequestErrorData struct {
	Kind    string `json:"kind,omitempty"`
	Message string `json:"message"`
}

type OpenCodeBadRequestError struct {
	Data OpenCodeBadRequestErrorData `json:"data"`
	Name string                      `json:"name"`
}

type OpenCodeCatalogUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeCatalogUpdated struct {
	Data     map[string]any                 `json:"data"`
	Durable  *OpenCodeCatalogUpdatedDurable `json:"durable,omitempty"`
	ID       string                         `json:"id"`
	Location OpenCodeLocationRef            `json:"location,omitempty"`
	Metadata map[string]any                 `json:"metadata,omitempty"`
	Type     string                         `json:"type"`
}

type OpenCodeCommand struct {
	Agent       string   `json:"agent,omitempty"`
	Description string   `json:"description,omitempty"`
	Hints       []string `json:"hints"`
	Model       string   `json:"model,omitempty"`
	Name        string   `json:"name"`
	Source      string   `json:"source,omitempty"`
	Subtask     bool     `json:"subtask,omitempty"`
	Template    string   `json:"template"`
}

type OpenCodeCommandExecutedData struct {
	Arguments string `json:"arguments"`
	MessageID string `json:"messageID"`
	Name      string `json:"name"`
	SessionID string `json:"sessionID"`
}

type OpenCodeCommandExecutedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeCommandExecuted struct {
	Data     OpenCodeCommandExecutedData     `json:"data"`
	Durable  *OpenCodeCommandExecutedDurable `json:"durable,omitempty"`
	ID       string                          `json:"id"`
	Location OpenCodeLocationRef             `json:"location,omitempty"`
	Metadata map[string]any                  `json:"metadata,omitempty"`
	Type     string                          `json:"type"`
}

type OpenCodeCommandV2Info struct {
	Agent       string           `json:"agent,omitempty"`
	Description string           `json:"description,omitempty"`
	Model       OpenCodeModelRef `json:"model,omitempty"`
	Name        string           `json:"name"`
	Subtask     bool             `json:"subtask,omitempty"`
	Template    string           `json:"template"`
}

type OpenCodeCompactionPart struct {
	Auto        bool   `json:"auto"`
	ID          string `json:"id"`
	MessageID   string `json:"messageID"`
	Overflow    bool   `json:"overflow,omitempty"`
	SessionID   string `json:"sessionID"`
	TailStartId string `json:"tail_start_id,omitempty"`
	Type        string `json:"type"`
}

type OpenCodeConfigAgent struct {
	Build      OpenCodeAgentConfig `json:"build,omitempty"`
	Compaction OpenCodeAgentConfig `json:"compaction,omitempty"`
	Explore    OpenCodeAgentConfig `json:"explore,omitempty"`
	General    OpenCodeAgentConfig `json:"general,omitempty"`
	Plan       OpenCodeAgentConfig `json:"plan,omitempty"`
	Summary    OpenCodeAgentConfig `json:"summary,omitempty"`
	Title      OpenCodeAgentConfig `json:"title,omitempty"`
}

type OpenCodeConfigCompaction struct {
	Auto                 bool  `json:"auto,omitempty"`
	PreserveRecentTokens int64 `json:"preserve_recent_tokens,omitempty"`
	Prune                bool  `json:"prune,omitempty"`
	Reserved             int64 `json:"reserved,omitempty"`
	TailTurns            int64 `json:"tail_turns,omitempty"`
}

type OpenCodeConfigEnterprise struct {
	Url string `json:"url,omitempty"`
}

type OpenCodeConfigExperimental struct {
	BatchTool           bool                                 `json:"batch_tool,omitempty"`
	ContinueLoopOnDeny  bool                                 `json:"continue_loop_on_deny,omitempty"`
	DisablePasteSummary bool                                 `json:"disable_paste_summary,omitempty"`
	McpTimeout          int64                                `json:"mcp_timeout,omitempty"`
	OpenTelemetry       bool                                 `json:"openTelemetry,omitempty"`
	Policies            []OpenCodeConfigV2ExperimentalPolicy `json:"policies,omitempty"`
	PrimaryTools        []string                             `json:"primary_tools,omitempty"`
}

type OpenCodeConfigMode struct {
	Build OpenCodeAgentConfig `json:"build,omitempty"`
	Plan  OpenCodeAgentConfig `json:"plan,omitempty"`
}

type OpenCodeConfigSkills struct {
	Paths []string `json:"paths,omitempty"`
	Urls  []string `json:"urls,omitempty"`
}

type OpenCodeConfigToolOutput struct {
	MaxBytes int64 `json:"max_bytes,omitempty"`
	MaxLines int64 `json:"max_lines,omitempty"`
}

type OpenCodeConfigWatcher struct {
	Ignore []string `json:"ignore,omitempty"`
}

type OpenCodeConfig struct {
	Schema     string                   `json:"$schema,omitempty"`
	Agent      *OpenCodeConfigAgent     `json:"agent,omitempty"`
	Attachment OpenCodeAttachmentConfig `json:"attachment,omitempty"`
	Autoshare  bool                     `json:"autoshare,omitempty"`
	Autoupdate any                      `json:"autoupdate,omitempty"`
	Command    map[string]struct {
		Agent       string `json:"agent,omitempty"`
		Description string `json:"description,omitempty"`
		Model       string `json:"model,omitempty"`
		Subtask     bool   `json:"subtask,omitempty"`
		Template    string `json:"template"`
		Variant     string `json:"variant,omitempty"`
	} `json:"command,omitempty"`
	Compaction        *OpenCodeConfigCompaction         `json:"compaction,omitempty"`
	DefaultAgent      string                            `json:"default_agent,omitempty"`
	DisabledProviders []string                          `json:"disabled_providers,omitempty"`
	EnabledProviders  []string                          `json:"enabled_providers,omitempty"`
	Enterprise        *OpenCodeConfigEnterprise         `json:"enterprise,omitempty"`
	Experimental      *OpenCodeConfigExperimental       `json:"experimental,omitempty"`
	Formatter         any                               `json:"formatter,omitempty"`
	Instructions      []string                          `json:"instructions,omitempty"`
	Layout            OpenCodeLayoutConfig              `json:"layout,omitempty"`
	LogLevel          OpenCodeLogLevel                  `json:"logLevel,omitempty"`
	Lsp               any                               `json:"lsp,omitempty"`
	Mcp               map[string]any                    `json:"mcp,omitempty"`
	Mode              *OpenCodeConfigMode               `json:"mode,omitempty"`
	Model             string                            `json:"model,omitempty"`
	Permission        OpenCodePermissionConfig          `json:"permission,omitempty"`
	Plugin            []any                             `json:"plugin,omitempty"`
	Provider          map[string]OpenCodeProviderConfig `json:"provider,omitempty"`
	Reference         map[string]any                    `json:"reference,omitempty"`
	References        map[string]any                    `json:"references,omitempty"`
	Server            OpenCodeServerConfig              `json:"server,omitempty"`
	Share             string                            `json:"share,omitempty"`
	Shell             string                            `json:"shell,omitempty"`
	Skills            *OpenCodeConfigSkills             `json:"skills,omitempty"`
	SmallModel        string                            `json:"small_model,omitempty"`
	Snapshot          bool                              `json:"snapshot,omitempty"`
	SubagentDepth     int64                             `json:"subagent_depth,omitempty"`
	ToolOutput        *OpenCodeConfigToolOutput         `json:"tool_output,omitempty"`
	Tools             map[string]bool                   `json:"tools,omitempty"`
	Username          string                            `json:"username,omitempty"`
	Watcher           *OpenCodeConfigWatcher            `json:"watcher,omitempty"`
}

type OpenCodeConfigV2ExperimentalPolicy struct {
	Action   string               `json:"action"`
	Effect   OpenCodePolicyEffect `json:"effect"`
	Resource string               `json:"resource"`
}

type OpenCodeConfigV2ReferenceGit struct {
	Branch      string `json:"branch,omitempty"`
	Description string `json:"description,omitempty"`
	Hidden      bool   `json:"hidden,omitempty"`
	Repository  string `json:"repository"`
}

type OpenCodeConfigV2ReferenceLocal struct {
	Description string `json:"description,omitempty"`
	Hidden      bool   `json:"hidden,omitempty"`
	Path        string `json:"path"`
}

type OpenCodeConflictError struct {
	Tag      string `json:"_tag"`
	Message  string `json:"message"`
	Resource string `json:"resource,omitempty"`
}

type OpenCodeConnectionCredentialInfo struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

type OpenCodeConnectionEnvInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type OpenCodeConnectionInfo any

type OpenCodeConsoleState struct {
	ActiveOrgName           string   `json:"activeOrgName,omitempty"`
	ConsoleManagedProviders []string `json:"consoleManagedProviders"`
	SwitchableOrgCount      int64    `json:"switchableOrgCount"`
}

type OpenCodeContentFilterErrorData struct {
	Message string `json:"message"`
}

type OpenCodeContentFilterError struct {
	Data OpenCodeContentFilterErrorData `json:"data"`
	Name string                         `json:"name"`
}

type OpenCodeContextOverflowErrorData struct {
	Message      string `json:"message"`
	ResponseBody string `json:"responseBody,omitempty"`
}

type OpenCodeContextOverflowError struct {
	Data OpenCodeContextOverflowErrorData `json:"data"`
	Name string                           `json:"name"`
}

type OpenCodeCredentialKey struct {
	Key      string         `json:"key"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Type     string         `json:"type"`
}

type OpenCodeCredentialOAuth struct {
	Access   string         `json:"access"`
	Expires  int64          `json:"expires"`
	Metadata map[string]any `json:"metadata,omitempty"`
	MethodID string         `json:"methodID"`
	Refresh  string         `json:"refresh"`
	Type     string         `json:"type"`
}

type OpenCodeCredentialValue any

type OpenCodeEvent any

type OpenCodeEventTuiCommandExecuteProperties struct {
	Command any `json:"command"`
}

type OpenCodeEventTuiCommandExecute struct {
	ID         string                                   `json:"id"`
	Properties OpenCodeEventTuiCommandExecuteProperties `json:"properties"`
	Type       string                                   `json:"type"`
}

type OpenCodeEventTuiPromptAppendProperties struct {
	Text string `json:"text"`
}

type OpenCodeEventTuiPromptAppend struct {
	ID         string                                 `json:"id"`
	Properties OpenCodeEventTuiPromptAppendProperties `json:"properties"`
	Type       string                                 `json:"type"`
}

type OpenCodeEventTuiSessionSelectProperties struct {
	SessionID string `json:"sessionID"`
}

type OpenCodeEventTuiSessionSelect struct {
	ID         string                                  `json:"id"`
	Properties OpenCodeEventTuiSessionSelectProperties `json:"properties"`
	Type       string                                  `json:"type"`
}

type OpenCodeEventTuiToastShowProperties struct {
	Duration int64  `json:"duration,omitempty"`
	Message  string `json:"message"`
	Title    string `json:"title,omitempty"`
	Variant  string `json:"variant"`
}

type OpenCodeEventTuiToastShow struct {
	ID         string                              `json:"id"`
	Properties OpenCodeEventTuiToastShowProperties `json:"properties"`
	Type       string                              `json:"type"`
}

type OpenCodeEventCatalogUpdated struct {
	ID         string         `json:"id"`
	Properties map[string]any `json:"properties"`
	Type       string         `json:"type"`
}

type OpenCodeEventCommandExecutedProperties struct {
	Arguments string `json:"arguments"`
	MessageID string `json:"messageID"`
	Name      string `json:"name"`
	SessionID string `json:"sessionID"`
}

type OpenCodeEventCommandExecuted struct {
	ID         string                                 `json:"id"`
	Properties OpenCodeEventCommandExecutedProperties `json:"properties"`
	Type       string                                 `json:"type"`
}

type OpenCodeEventFileEditedProperties struct {
	File string `json:"file"`
}

type OpenCodeEventFileEdited struct {
	ID         string                            `json:"id"`
	Properties OpenCodeEventFileEditedProperties `json:"properties"`
	Type       string                            `json:"type"`
}

type OpenCodeEventFileWatcherUpdatedProperties struct {
	Event string `json:"event"`
	File  string `json:"file"`
}

type OpenCodeEventFileWatcherUpdated struct {
	ID         string                                    `json:"id"`
	Properties OpenCodeEventFileWatcherUpdatedProperties `json:"properties"`
	Type       string                                    `json:"type"`
}

type OpenCodeEventGlobalDisposed struct {
	ID         string         `json:"id"`
	Properties map[string]any `json:"properties"`
	Type       string         `json:"type"`
}

type OpenCodeEventInstallationUpdateAvailableProperties struct {
	Version string `json:"version"`
}

type OpenCodeEventInstallationUpdateAvailable struct {
	ID         string                                             `json:"id"`
	Properties OpenCodeEventInstallationUpdateAvailableProperties `json:"properties"`
	Type       string                                             `json:"type"`
}

type OpenCodeEventInstallationUpdatedProperties struct {
	Version string `json:"version"`
}

type OpenCodeEventInstallationUpdated struct {
	ID         string                                     `json:"id"`
	Properties OpenCodeEventInstallationUpdatedProperties `json:"properties"`
	Type       string                                     `json:"type"`
}

type OpenCodeEventIntegrationConnectionUpdatedProperties struct {
	IntegrationID string `json:"integrationID"`
}

type OpenCodeEventIntegrationConnectionUpdated struct {
	ID         string                                              `json:"id"`
	Properties OpenCodeEventIntegrationConnectionUpdatedProperties `json:"properties"`
	Type       string                                              `json:"type"`
}

type OpenCodeEventIntegrationUpdated struct {
	ID         string         `json:"id"`
	Properties map[string]any `json:"properties"`
	Type       string         `json:"type"`
}

type OpenCodeEventLspUpdated struct {
	ID         string         `json:"id"`
	Properties map[string]any `json:"properties"`
	Type       string         `json:"type"`
}

type OpenCodeEventMcpBrowserOpenFailedProperties struct {
	McpName string `json:"mcpName"`
	Url     string `json:"url"`
}

type OpenCodeEventMcpBrowserOpenFailed struct {
	ID         string                                      `json:"id"`
	Properties OpenCodeEventMcpBrowserOpenFailedProperties `json:"properties"`
	Type       string                                      `json:"type"`
}

type OpenCodeEventMcpToolsChangedProperties struct {
	Server string `json:"server"`
}

type OpenCodeEventMcpToolsChanged struct {
	ID         string                                 `json:"id"`
	Properties OpenCodeEventMcpToolsChangedProperties `json:"properties"`
	Type       string                                 `json:"type"`
}

type OpenCodeEventMessagePartDeltaProperties struct {
	Delta     string `json:"delta"`
	Field     string `json:"field"`
	MessageID string `json:"messageID"`
	PartID    string `json:"partID"`
	SessionID string `json:"sessionID"`
}

type OpenCodeEventMessagePartDelta struct {
	ID         string                                  `json:"id"`
	Properties OpenCodeEventMessagePartDeltaProperties `json:"properties"`
	Type       string                                  `json:"type"`
}

type OpenCodeEventMessagePartRemovedProperties struct {
	MessageID string `json:"messageID"`
	PartID    string `json:"partID"`
	SessionID string `json:"sessionID"`
}

type OpenCodeEventMessagePartRemoved struct {
	ID         string                                    `json:"id"`
	Properties OpenCodeEventMessagePartRemovedProperties `json:"properties"`
	Type       string                                    `json:"type"`
}

type OpenCodeEventMessagePartUpdatedProperties struct {
	Part      OpenCodePart `json:"part"`
	SessionID string       `json:"sessionID"`
	Time      float64      `json:"time"`
}

type OpenCodeEventMessagePartUpdated struct {
	ID         string                                    `json:"id"`
	Properties OpenCodeEventMessagePartUpdatedProperties `json:"properties"`
	Type       string                                    `json:"type"`
}

type OpenCodeEventMessageRemovedProperties struct {
	MessageID string `json:"messageID"`
	SessionID string `json:"sessionID"`
}

type OpenCodeEventMessageRemoved struct {
	ID         string                                `json:"id"`
	Properties OpenCodeEventMessageRemovedProperties `json:"properties"`
	Type       string                                `json:"type"`
}

type OpenCodeEventMessageUpdatedProperties struct {
	Info      OpenCodeMessage `json:"info"`
	SessionID string          `json:"sessionID"`
}

type OpenCodeEventMessageUpdated struct {
	ID         string                                `json:"id"`
	Properties OpenCodeEventMessageUpdatedProperties `json:"properties"`
	Type       string                                `json:"type"`
}

type OpenCodeEventModelsDevRefreshed struct {
	ID         string         `json:"id"`
	Properties map[string]any `json:"properties"`
	Type       string         `json:"type"`
}

type OpenCodeEventPermissionAskedPropertiesTool struct {
	CallID    string `json:"callID"`
	MessageID string `json:"messageID"`
}

type OpenCodeEventPermissionAskedProperties struct {
	Always     []string                                    `json:"always"`
	ID         string                                      `json:"id"`
	Metadata   map[string]any                              `json:"metadata"`
	Patterns   []string                                    `json:"patterns"`
	Permission string                                      `json:"permission"`
	SessionID  string                                      `json:"sessionID"`
	Tool       *OpenCodeEventPermissionAskedPropertiesTool `json:"tool,omitempty"`
}

type OpenCodeEventPermissionAsked struct {
	ID         string                                 `json:"id"`
	Properties OpenCodeEventPermissionAskedProperties `json:"properties"`
	Type       string                                 `json:"type"`
}

type OpenCodeEventPermissionRepliedProperties struct {
	Reply     string `json:"reply"`
	RequestID string `json:"requestID"`
	SessionID string `json:"sessionID"`
}

type OpenCodeEventPermissionReplied struct {
	ID         string                                   `json:"id"`
	Properties OpenCodeEventPermissionRepliedProperties `json:"properties"`
	Type       string                                   `json:"type"`
}

type OpenCodeEventPermissionV2AskedProperties struct {
	Action    string                     `json:"action"`
	ID        string                     `json:"id"`
	Metadata  map[string]any             `json:"metadata,omitempty"`
	Resources []string                   `json:"resources"`
	Save      []string                   `json:"save,omitempty"`
	SessionID string                     `json:"sessionID"`
	Source    OpenCodePermissionV2Source `json:"source,omitempty"`
}

type OpenCodeEventPermissionV2Asked struct {
	ID         string                                   `json:"id"`
	Properties OpenCodeEventPermissionV2AskedProperties `json:"properties"`
	Type       string                                   `json:"type"`
}

type OpenCodeEventPermissionV2RepliedProperties struct {
	Reply     OpenCodePermissionV2Reply `json:"reply"`
	RequestID string                    `json:"requestID"`
	SessionID string                    `json:"sessionID"`
}

type OpenCodeEventPermissionV2Replied struct {
	ID         string                                     `json:"id"`
	Properties OpenCodeEventPermissionV2RepliedProperties `json:"properties"`
	Type       string                                     `json:"type"`
}

type OpenCodeEventPluginAddedProperties struct {
	ID string `json:"id"`
}

type OpenCodeEventPluginAdded struct {
	ID         string                             `json:"id"`
	Properties OpenCodeEventPluginAddedProperties `json:"properties"`
	Type       string                             `json:"type"`
}

type OpenCodeEventProjectDirectoriesUpdatedProperties struct {
	ProjectID string `json:"projectID"`
}

type OpenCodeEventProjectDirectoriesUpdated struct {
	ID         string                                           `json:"id"`
	Properties OpenCodeEventProjectDirectoriesUpdatedProperties `json:"properties"`
	Type       string                                           `json:"type"`
}

type OpenCodeEventProjectUpdatedProperties struct {
	Commands  OpenCodeProjectCommands `json:"commands,omitempty"`
	Icon      OpenCodeProjectIcon     `json:"icon,omitempty"`
	ID        string                  `json:"id"`
	Name      string                  `json:"name,omitempty"`
	Sandboxes []string                `json:"sandboxes"`
	Time      OpenCodeProjectTime     `json:"time"`
	Vcs       OpenCodeProjectVcs      `json:"vcs,omitempty"`
	Worktree  string                  `json:"worktree"`
}

type OpenCodeEventProjectUpdated struct {
	ID         string                                `json:"id"`
	Properties OpenCodeEventProjectUpdatedProperties `json:"properties"`
	Type       string                                `json:"type"`
}

type OpenCodeEventPtyCreatedProperties struct {
	Info OpenCodePty `json:"info"`
}

type OpenCodeEventPtyCreated struct {
	ID         string                            `json:"id"`
	Properties OpenCodeEventPtyCreatedProperties `json:"properties"`
	Type       string                            `json:"type"`
}

type OpenCodeEventPtyDeletedProperties struct {
	ID string `json:"id"`
}

type OpenCodeEventPtyDeleted struct {
	ID         string                            `json:"id"`
	Properties OpenCodeEventPtyDeletedProperties `json:"properties"`
	Type       string                            `json:"type"`
}

type OpenCodeEventPtyExitedProperties struct {
	ExitCode int64  `json:"exitCode"`
	ID       string `json:"id"`
}

type OpenCodeEventPtyExited struct {
	ID         string                           `json:"id"`
	Properties OpenCodeEventPtyExitedProperties `json:"properties"`
	Type       string                           `json:"type"`
}

type OpenCodeEventPtyUpdatedProperties struct {
	Info OpenCodePty `json:"info"`
}

type OpenCodeEventPtyUpdated struct {
	ID         string                            `json:"id"`
	Properties OpenCodeEventPtyUpdatedProperties `json:"properties"`
	Type       string                            `json:"type"`
}

type OpenCodeEventQuestionAskedProperties struct {
	ID        string                 `json:"id"`
	Questions []OpenCodeQuestionInfo `json:"questions"`
	SessionID string                 `json:"sessionID"`
	Tool      OpenCodeQuestionTool   `json:"tool,omitempty"`
}

type OpenCodeEventQuestionAsked struct {
	ID         string                               `json:"id"`
	Properties OpenCodeEventQuestionAskedProperties `json:"properties"`
	Type       string                               `json:"type"`
}

type OpenCodeEventQuestionRejectedProperties struct {
	RequestID string `json:"requestID"`
	SessionID string `json:"sessionID"`
}

type OpenCodeEventQuestionRejected struct {
	ID         string                                  `json:"id"`
	Properties OpenCodeEventQuestionRejectedProperties `json:"properties"`
	Type       string                                  `json:"type"`
}

type OpenCodeEventQuestionRepliedProperties struct {
	Answers   []OpenCodeQuestionAnswer `json:"answers"`
	RequestID string                   `json:"requestID"`
	SessionID string                   `json:"sessionID"`
}

type OpenCodeEventQuestionReplied struct {
	ID         string                                 `json:"id"`
	Properties OpenCodeEventQuestionRepliedProperties `json:"properties"`
	Type       string                                 `json:"type"`
}

type OpenCodeEventQuestionV2AskedProperties struct {
	ID        string                   `json:"id"`
	Questions []OpenCodeQuestionV2Info `json:"questions"`
	SessionID string                   `json:"sessionID"`
	Tool      OpenCodeQuestionV2Tool   `json:"tool,omitempty"`
}

type OpenCodeEventQuestionV2Asked struct {
	ID         string                                 `json:"id"`
	Properties OpenCodeEventQuestionV2AskedProperties `json:"properties"`
	Type       string                                 `json:"type"`
}

type OpenCodeEventQuestionV2RejectedProperties struct {
	RequestID string `json:"requestID"`
	SessionID string `json:"sessionID"`
}

type OpenCodeEventQuestionV2Rejected struct {
	ID         string                                    `json:"id"`
	Properties OpenCodeEventQuestionV2RejectedProperties `json:"properties"`
	Type       string                                    `json:"type"`
}

type OpenCodeEventQuestionV2RepliedProperties struct {
	Answers   []OpenCodeQuestionV2Answer `json:"answers"`
	RequestID string                     `json:"requestID"`
	SessionID string                     `json:"sessionID"`
}

type OpenCodeEventQuestionV2Replied struct {
	ID         string                                   `json:"id"`
	Properties OpenCodeEventQuestionV2RepliedProperties `json:"properties"`
	Type       string                                   `json:"type"`
}

type OpenCodeEventReferenceUpdated struct {
	ID         string         `json:"id"`
	Properties map[string]any `json:"properties"`
	Type       string         `json:"type"`
}

type OpenCodeEventServerConnected struct {
	ID         string         `json:"id"`
	Properties map[string]any `json:"properties"`
	Type       string         `json:"type"`
}

type OpenCodeEventServerInstanceDisposedProperties struct {
	Directory string `json:"directory"`
}

type OpenCodeEventServerInstanceDisposed struct {
	ID         string                                        `json:"id"`
	Properties OpenCodeEventServerInstanceDisposedProperties `json:"properties"`
	Type       string                                        `json:"type"`
}

type OpenCodeEventSessionCompactedProperties struct {
	SessionID string `json:"sessionID"`
}

type OpenCodeEventSessionCompacted struct {
	ID         string                                  `json:"id"`
	Properties OpenCodeEventSessionCompactedProperties `json:"properties"`
	Type       string                                  `json:"type"`
}

type OpenCodeEventSessionCreatedProperties struct {
	Info      OpenCodeSession `json:"info"`
	SessionID string          `json:"sessionID"`
}

type OpenCodeEventSessionCreated struct {
	ID         string                                `json:"id"`
	Properties OpenCodeEventSessionCreatedProperties `json:"properties"`
	Type       string                                `json:"type"`
}

type OpenCodeEventSessionDeletedProperties struct {
	Info      OpenCodeSession `json:"info"`
	SessionID string          `json:"sessionID"`
}

type OpenCodeEventSessionDeleted struct {
	ID         string                                `json:"id"`
	Properties OpenCodeEventSessionDeletedProperties `json:"properties"`
	Type       string                                `json:"type"`
}

type OpenCodeEventSessionDiffProperties struct {
	Diff      []OpenCodeSnapshotFileDiff `json:"diff"`
	SessionID string                     `json:"sessionID"`
}

type OpenCodeEventSessionDiff struct {
	ID         string                             `json:"id"`
	Properties OpenCodeEventSessionDiffProperties `json:"properties"`
	Type       string                             `json:"type"`
}

type OpenCodeEventSessionErrorProperties struct {
	Error     any    `json:"error,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
}

type OpenCodeEventSessionError struct {
	ID         string                              `json:"id"`
	Properties OpenCodeEventSessionErrorProperties `json:"properties"`
	Type       string                              `json:"type"`
}

type OpenCodeEventSessionIdleProperties struct {
	SessionID string `json:"sessionID"`
}

type OpenCodeEventSessionIdle struct {
	ID         string                             `json:"id"`
	Properties OpenCodeEventSessionIdleProperties `json:"properties"`
	Type       string                             `json:"type"`
}

type OpenCodeEventSessionNextAgentSwitchedProperties struct {
	Agent     string  `json:"agent"`
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextAgentSwitched struct {
	ID         string                                          `json:"id"`
	Properties OpenCodeEventSessionNextAgentSwitchedProperties `json:"properties"`
	Type       string                                          `json:"type"`
}

type OpenCodeEventSessionNextCompactionDeltaProperties struct {
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextCompactionDelta struct {
	ID         string                                            `json:"id"`
	Properties OpenCodeEventSessionNextCompactionDeltaProperties `json:"properties"`
	Type       string                                            `json:"type"`
}

type OpenCodeEventSessionNextCompactionEndedProperties struct {
	MessageID string  `json:"messageID"`
	Reason    string  `json:"reason"`
	Recent    string  `json:"recent"`
	SessionID string  `json:"sessionID"`
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextCompactionEnded struct {
	ID         string                                            `json:"id"`
	Properties OpenCodeEventSessionNextCompactionEndedProperties `json:"properties"`
	Type       string                                            `json:"type"`
}

type OpenCodeEventSessionNextCompactionStartedProperties struct {
	MessageID string  `json:"messageID"`
	Reason    string  `json:"reason"`
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextCompactionStarted struct {
	ID         string                                              `json:"id"`
	Properties OpenCodeEventSessionNextCompactionStartedProperties `json:"properties"`
	Type       string                                              `json:"type"`
}

type OpenCodeEventSessionNextContextUpdatedProperties struct {
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextContextUpdated struct {
	ID         string                                           `json:"id"`
	Properties OpenCodeEventSessionNextContextUpdatedProperties `json:"properties"`
	Type       string                                           `json:"type"`
}

type OpenCodeEventSessionNextModelSwitchedProperties struct {
	MessageID string           `json:"messageID"`
	Model     OpenCodeModelRef `json:"model"`
	SessionID string           `json:"sessionID"`
	Timestamp float64          `json:"timestamp"`
}

type OpenCodeEventSessionNextModelSwitched struct {
	ID         string                                          `json:"id"`
	Properties OpenCodeEventSessionNextModelSwitchedProperties `json:"properties"`
	Type       string                                          `json:"type"`
}

type OpenCodeEventSessionNextMovedProperties struct {
	Location     OpenCodeLocationRef `json:"location"`
	SessionID    string              `json:"sessionID"`
	Subdirectory string              `json:"subdirectory,omitempty"`
	Timestamp    float64             `json:"timestamp"`
}

type OpenCodeEventSessionNextMoved struct {
	ID         string                                  `json:"id"`
	Properties OpenCodeEventSessionNextMovedProperties `json:"properties"`
	Type       string                                  `json:"type"`
}

type OpenCodeEventSessionNextPromptAdmittedProperties struct {
	Delivery  string         `json:"delivery"`
	MessageID string         `json:"messageID"`
	Prompt    OpenCodePrompt `json:"prompt"`
	SessionID string         `json:"sessionID"`
	Timestamp float64        `json:"timestamp"`
}

type OpenCodeEventSessionNextPromptAdmitted struct {
	ID         string                                           `json:"id"`
	Properties OpenCodeEventSessionNextPromptAdmittedProperties `json:"properties"`
	Type       string                                           `json:"type"`
}

type OpenCodeEventSessionNextPromptedProperties struct {
	Delivery  string         `json:"delivery"`
	MessageID string         `json:"messageID"`
	Prompt    OpenCodePrompt `json:"prompt"`
	SessionID string         `json:"sessionID"`
	Timestamp float64        `json:"timestamp"`
}

type OpenCodeEventSessionNextPrompted struct {
	ID         string                                     `json:"id"`
	Properties OpenCodeEventSessionNextPromptedProperties `json:"properties"`
	Type       string                                     `json:"type"`
}

type OpenCodeEventSessionNextReasoningDeltaProperties struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	Delta              string  `json:"delta"`
	ReasoningID        string  `json:"reasoningID"`
	SessionID          string  `json:"sessionID"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextReasoningDelta struct {
	ID         string                                           `json:"id"`
	Properties OpenCodeEventSessionNextReasoningDeltaProperties `json:"properties"`
	Type       string                                           `json:"type"`
}

type OpenCodeEventSessionNextReasoningEndedProperties struct {
	AssistantMessageID string                      `json:"assistantMessageID"`
	ProviderMetadata   OpenCodeLLMProviderMetadata `json:"providerMetadata,omitempty"`
	ReasoningID        string                      `json:"reasoningID"`
	SessionID          string                      `json:"sessionID"`
	Text               string                      `json:"text"`
	Timestamp          float64                     `json:"timestamp"`
}

type OpenCodeEventSessionNextReasoningEnded struct {
	ID         string                                           `json:"id"`
	Properties OpenCodeEventSessionNextReasoningEndedProperties `json:"properties"`
	Type       string                                           `json:"type"`
}

type OpenCodeEventSessionNextReasoningStartedProperties struct {
	AssistantMessageID string                      `json:"assistantMessageID"`
	ProviderMetadata   OpenCodeLLMProviderMetadata `json:"providerMetadata,omitempty"`
	ReasoningID        string                      `json:"reasoningID"`
	SessionID          string                      `json:"sessionID"`
	Timestamp          float64                     `json:"timestamp"`
}

type OpenCodeEventSessionNextReasoningStarted struct {
	ID         string                                             `json:"id"`
	Properties OpenCodeEventSessionNextReasoningStartedProperties `json:"properties"`
	Type       string                                             `json:"type"`
}

type OpenCodeEventSessionNextRetriedProperties struct {
	Attempt   float64                       `json:"attempt"`
	Error     OpenCodeSessionNextRetryError `json:"error"`
	SessionID string                        `json:"sessionID"`
	Timestamp float64                       `json:"timestamp"`
}

type OpenCodeEventSessionNextRetried struct {
	ID         string                                    `json:"id"`
	Properties OpenCodeEventSessionNextRetriedProperties `json:"properties"`
	Type       string                                    `json:"type"`
}

type OpenCodeEventSessionNextRevertClearedProperties struct {
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextRevertCleared struct {
	ID         string                                          `json:"id"`
	Properties OpenCodeEventSessionNextRevertClearedProperties `json:"properties"`
	Type       string                                          `json:"type"`
}

type OpenCodeEventSessionNextRevertCommittedProperties struct {
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextRevertCommitted struct {
	ID         string                                            `json:"id"`
	Properties OpenCodeEventSessionNextRevertCommittedProperties `json:"properties"`
	Type       string                                            `json:"type"`
}

type OpenCodeEventSessionNextRevertStagedProperties struct {
	Revert    OpenCodeRevertState `json:"revert"`
	SessionID string              `json:"sessionID"`
	Timestamp float64             `json:"timestamp"`
}

type OpenCodeEventSessionNextRevertStaged struct {
	ID         string                                         `json:"id"`
	Properties OpenCodeEventSessionNextRevertStagedProperties `json:"properties"`
	Type       string                                         `json:"type"`
}

type OpenCodeEventSessionNextShellEndedProperties struct {
	CallID    string  `json:"callID"`
	Output    string  `json:"output"`
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextShellEnded struct {
	ID         string                                       `json:"id"`
	Properties OpenCodeEventSessionNextShellEndedProperties `json:"properties"`
	Type       string                                       `json:"type"`
}

type OpenCodeEventSessionNextShellStartedProperties struct {
	CallID    string  `json:"callID"`
	Command   string  `json:"command"`
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextShellStarted struct {
	ID         string                                         `json:"id"`
	Properties OpenCodeEventSessionNextShellStartedProperties `json:"properties"`
	Type       string                                         `json:"type"`
}

type OpenCodeEventSessionNextStepEndedPropertiesTokensCache struct {
	Read  float64 `json:"read"`
	Write float64 `json:"write"`
}

type OpenCodeEventSessionNextStepEndedPropertiesTokens struct {
	Cache     OpenCodeEventSessionNextStepEndedPropertiesTokensCache `json:"cache"`
	Input     float64                                                `json:"input"`
	Output    float64                                                `json:"output"`
	Reasoning float64                                                `json:"reasoning"`
}

type OpenCodeEventSessionNextStepEndedProperties struct {
	AssistantMessageID string                                            `json:"assistantMessageID"`
	Cost               float64                                           `json:"cost"`
	Files              []string                                          `json:"files,omitempty"`
	Finish             string                                            `json:"finish"`
	SessionID          string                                            `json:"sessionID"`
	Snapshot           string                                            `json:"snapshot,omitempty"`
	Timestamp          float64                                           `json:"timestamp"`
	Tokens             OpenCodeEventSessionNextStepEndedPropertiesTokens `json:"tokens"`
}

type OpenCodeEventSessionNextStepEnded struct {
	ID         string                                      `json:"id"`
	Properties OpenCodeEventSessionNextStepEndedProperties `json:"properties"`
	Type       string                                      `json:"type"`
}

type OpenCodeEventSessionNextStepFailedProperties struct {
	AssistantMessageID string                      `json:"assistantMessageID"`
	Error              OpenCodeSessionErrorUnknown `json:"error"`
	SessionID          string                      `json:"sessionID"`
	Timestamp          float64                     `json:"timestamp"`
}

type OpenCodeEventSessionNextStepFailed struct {
	ID         string                                       `json:"id"`
	Properties OpenCodeEventSessionNextStepFailedProperties `json:"properties"`
	Type       string                                       `json:"type"`
}

type OpenCodeEventSessionNextStepStartedProperties struct {
	Agent              string           `json:"agent"`
	AssistantMessageID string           `json:"assistantMessageID"`
	Model              OpenCodeModelRef `json:"model"`
	SessionID          string           `json:"sessionID"`
	Snapshot           string           `json:"snapshot,omitempty"`
	Timestamp          float64          `json:"timestamp"`
}

type OpenCodeEventSessionNextStepStarted struct {
	ID         string                                        `json:"id"`
	Properties OpenCodeEventSessionNextStepStartedProperties `json:"properties"`
	Type       string                                        `json:"type"`
}

type OpenCodeEventSessionNextSyntheticProperties struct {
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextSynthetic struct {
	ID         string                                      `json:"id"`
	Properties OpenCodeEventSessionNextSyntheticProperties `json:"properties"`
	Type       string                                      `json:"type"`
}

type OpenCodeEventSessionNextTextDeltaProperties struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	Delta              string  `json:"delta"`
	SessionID          string  `json:"sessionID"`
	TextID             string  `json:"textID"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextTextDelta struct {
	ID         string                                      `json:"id"`
	Properties OpenCodeEventSessionNextTextDeltaProperties `json:"properties"`
	Type       string                                      `json:"type"`
}

type OpenCodeEventSessionNextTextEndedProperties struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	SessionID          string  `json:"sessionID"`
	Text               string  `json:"text"`
	TextID             string  `json:"textID"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextTextEnded struct {
	ID         string                                      `json:"id"`
	Properties OpenCodeEventSessionNextTextEndedProperties `json:"properties"`
	Type       string                                      `json:"type"`
}

type OpenCodeEventSessionNextTextStartedProperties struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	SessionID          string  `json:"sessionID"`
	TextID             string  `json:"textID"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextTextStarted struct {
	ID         string                                        `json:"id"`
	Properties OpenCodeEventSessionNextTextStartedProperties `json:"properties"`
	Type       string                                        `json:"type"`
}

type OpenCodeEventSessionNextToolCalledPropertiesProvider struct {
	Executed bool                        `json:"executed"`
	Metadata OpenCodeLLMProviderMetadata `json:"metadata,omitempty"`
}

type OpenCodeEventSessionNextToolCalledProperties struct {
	AssistantMessageID string                                               `json:"assistantMessageID"`
	CallID             string                                               `json:"callID"`
	Input              map[string]any                                       `json:"input"`
	Provider           OpenCodeEventSessionNextToolCalledPropertiesProvider `json:"provider"`
	SessionID          string                                               `json:"sessionID"`
	Timestamp          float64                                              `json:"timestamp"`
	Tool               string                                               `json:"tool"`
}

type OpenCodeEventSessionNextToolCalled struct {
	ID         string                                       `json:"id"`
	Properties OpenCodeEventSessionNextToolCalledProperties `json:"properties"`
	Type       string                                       `json:"type"`
}

type OpenCodeEventSessionNextToolFailedPropertiesProvider struct {
	Executed bool                        `json:"executed"`
	Metadata OpenCodeLLMProviderMetadata `json:"metadata,omitempty"`
}

type OpenCodeEventSessionNextToolFailedProperties struct {
	AssistantMessageID string                                               `json:"assistantMessageID"`
	CallID             string                                               `json:"callID"`
	Error              OpenCodeSessionErrorUnknown                          `json:"error"`
	Provider           OpenCodeEventSessionNextToolFailedPropertiesProvider `json:"provider"`
	Result             any                                                  `json:"result,omitempty"`
	SessionID          string                                               `json:"sessionID"`
	Timestamp          float64                                              `json:"timestamp"`
}

type OpenCodeEventSessionNextToolFailed struct {
	ID         string                                       `json:"id"`
	Properties OpenCodeEventSessionNextToolFailedProperties `json:"properties"`
	Type       string                                       `json:"type"`
}

type OpenCodeEventSessionNextToolInputDeltaProperties struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	CallID             string  `json:"callID"`
	Delta              string  `json:"delta"`
	SessionID          string  `json:"sessionID"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextToolInputDelta struct {
	ID         string                                           `json:"id"`
	Properties OpenCodeEventSessionNextToolInputDeltaProperties `json:"properties"`
	Type       string                                           `json:"type"`
}

type OpenCodeEventSessionNextToolInputEndedProperties struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	CallID             string  `json:"callID"`
	SessionID          string  `json:"sessionID"`
	Text               string  `json:"text"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextToolInputEnded struct {
	ID         string                                           `json:"id"`
	Properties OpenCodeEventSessionNextToolInputEndedProperties `json:"properties"`
	Type       string                                           `json:"type"`
}

type OpenCodeEventSessionNextToolInputStartedProperties struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	CallID             string  `json:"callID"`
	Name               string  `json:"name"`
	SessionID          string  `json:"sessionID"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeEventSessionNextToolInputStarted struct {
	ID         string                                             `json:"id"`
	Properties OpenCodeEventSessionNextToolInputStartedProperties `json:"properties"`
	Type       string                                             `json:"type"`
}

type OpenCodeEventSessionNextToolProgressProperties struct {
	AssistantMessageID string                   `json:"assistantMessageID"`
	CallID             string                   `json:"callID"`
	Content            []OpenCodeLLMToolContent `json:"content"`
	SessionID          string                   `json:"sessionID"`
	Structured         map[string]any           `json:"structured"`
	Timestamp          float64                  `json:"timestamp"`
}

type OpenCodeEventSessionNextToolProgress struct {
	ID         string                                         `json:"id"`
	Properties OpenCodeEventSessionNextToolProgressProperties `json:"properties"`
	Type       string                                         `json:"type"`
}

type OpenCodeEventSessionNextToolSuccessPropertiesProvider struct {
	Executed bool                        `json:"executed"`
	Metadata OpenCodeLLMProviderMetadata `json:"metadata,omitempty"`
}

type OpenCodeEventSessionNextToolSuccessProperties struct {
	AssistantMessageID string                                                `json:"assistantMessageID"`
	CallID             string                                                `json:"callID"`
	Content            []OpenCodeLLMToolContent                              `json:"content"`
	OutputPaths        []string                                              `json:"outputPaths,omitempty"`
	Provider           OpenCodeEventSessionNextToolSuccessPropertiesProvider `json:"provider"`
	Result             any                                                   `json:"result,omitempty"`
	SessionID          string                                                `json:"sessionID"`
	Structured         map[string]any                                        `json:"structured"`
	Timestamp          float64                                               `json:"timestamp"`
}

type OpenCodeEventSessionNextToolSuccess struct {
	ID         string                                        `json:"id"`
	Properties OpenCodeEventSessionNextToolSuccessProperties `json:"properties"`
	Type       string                                        `json:"type"`
}

type OpenCodeEventSessionStatusProperties struct {
	SessionID string                `json:"sessionID"`
	Status    OpenCodeSessionStatus `json:"status"`
}

type OpenCodeEventSessionStatus struct {
	ID         string                               `json:"id"`
	Properties OpenCodeEventSessionStatusProperties `json:"properties"`
	Type       string                               `json:"type"`
}

type OpenCodeEventSessionUpdatedProperties struct {
	Info      OpenCodeSession `json:"info"`
	SessionID string          `json:"sessionID"`
}

type OpenCodeEventSessionUpdated struct {
	ID         string                                `json:"id"`
	Properties OpenCodeEventSessionUpdatedProperties `json:"properties"`
	Type       string                                `json:"type"`
}

type OpenCodeEventTodoUpdatedProperties struct {
	SessionID string         `json:"sessionID"`
	Todos     []OpenCodeTodo `json:"todos"`
}

type OpenCodeEventTodoUpdated struct {
	ID         string                             `json:"id"`
	Properties OpenCodeEventTodoUpdatedProperties `json:"properties"`
	Type       string                             `json:"type"`
}

type OpenCodeEventTuiCommandExecute2Properties struct {
	Command any `json:"command"`
}

type OpenCodeEventTuiCommandExecute2 struct {
	Properties OpenCodeEventTuiCommandExecute2Properties `json:"properties"`
	Type       string                                    `json:"type"`
}

type OpenCodeEventTuiPromptAppend2Properties struct {
	Text string `json:"text"`
}

type OpenCodeEventTuiPromptAppend2 struct {
	Properties OpenCodeEventTuiPromptAppend2Properties `json:"properties"`
	Type       string                                  `json:"type"`
}

type OpenCodeEventTuiSessionSelect2Properties struct {
	SessionID string `json:"sessionID"`
}

type OpenCodeEventTuiSessionSelect2 struct {
	Properties OpenCodeEventTuiSessionSelect2Properties `json:"properties"`
	Type       string                                   `json:"type"`
}

type OpenCodeEventTuiToastShow2Properties struct {
	Duration int64  `json:"duration,omitempty"`
	Message  string `json:"message"`
	Title    string `json:"title,omitempty"`
	Variant  string `json:"variant"`
}

type OpenCodeEventTuiToastShow2 struct {
	Properties OpenCodeEventTuiToastShow2Properties `json:"properties"`
	Type       string                               `json:"type"`
}

type OpenCodeEventVcsBranchUpdatedProperties struct {
	Branch string `json:"branch,omitempty"`
}

type OpenCodeEventVcsBranchUpdated struct {
	ID         string                                  `json:"id"`
	Properties OpenCodeEventVcsBranchUpdatedProperties `json:"properties"`
	Type       string                                  `json:"type"`
}

type OpenCodeEventWorkspaceFailedProperties struct {
	Message string `json:"message"`
}

type OpenCodeEventWorkspaceFailed struct {
	ID         string                                 `json:"id"`
	Properties OpenCodeEventWorkspaceFailedProperties `json:"properties"`
	Type       string                                 `json:"type"`
}

type OpenCodeEventWorkspaceReadyProperties struct {
	Name string `json:"name"`
}

type OpenCodeEventWorkspaceReady struct {
	ID         string                                `json:"id"`
	Properties OpenCodeEventWorkspaceReadyProperties `json:"properties"`
	Type       string                                `json:"type"`
}

type OpenCodeEventWorkspaceStatusProperties struct {
	Status      string `json:"status"`
	WorkspaceID string `json:"workspaceID"`
}

type OpenCodeEventWorkspaceStatus struct {
	ID         string                                 `json:"id"`
	Properties OpenCodeEventWorkspaceStatusProperties `json:"properties"`
	Type       string                                 `json:"type"`
}

type OpenCodeEventWorktreeFailedProperties struct {
	Message string `json:"message"`
}

type OpenCodeEventWorktreeFailed struct {
	ID         string                                `json:"id"`
	Properties OpenCodeEventWorktreeFailedProperties `json:"properties"`
	Type       string                                `json:"type"`
}

type OpenCodeEventWorktreeReadyProperties struct {
	Branch string `json:"branch,omitempty"`
	Name   string `json:"name"`
}

type OpenCodeEventWorktreeReady struct {
	ID         string                               `json:"id"`
	Properties OpenCodeEventWorktreeReadyProperties `json:"properties"`
	Type       string                               `json:"type"`
}

type OpenCodeExperimentalCapabilities struct {
	BackgroundSubagents bool `json:"backgroundSubagents"`
}

type OpenCodeFile struct {
	Added   int64  `json:"added"`
	Path    string `json:"path"`
	Removed int64  `json:"removed"`
	Status  string `json:"status"`
}

type OpenCodeFileContentPatchHunksItem struct {
	Lines    []string `json:"lines"`
	NewLines int64    `json:"newLines"`
	NewStart int64    `json:"newStart"`
	OldLines int64    `json:"oldLines"`
	OldStart int64    `json:"oldStart"`
}

type OpenCodeFileContentPatch struct {
	Hunks       []OpenCodeFileContentPatchHunksItem `json:"hunks"`
	Index       string                              `json:"index,omitempty"`
	NewFileName string                              `json:"newFileName"`
	NewHeader   string                              `json:"newHeader,omitempty"`
	OldFileName string                              `json:"oldFileName"`
	OldHeader   string                              `json:"oldHeader,omitempty"`
}

type OpenCodeFileContent struct {
	Content  string                    `json:"content"`
	Diff     string                    `json:"diff,omitempty"`
	Encoding string                    `json:"encoding,omitempty"`
	MimeType string                    `json:"mimeType,omitempty"`
	Patch    *OpenCodeFileContentPatch `json:"patch,omitempty"`
	Type     string                    `json:"type"`
}

type OpenCodeFileDiff struct {
	Additions int64  `json:"additions"`
	Deletions int64  `json:"deletions"`
	Patch     string `json:"patch"`
	Path      string `json:"path"`
	Status    string `json:"status"`
}

type OpenCodeFileEditedData struct {
	File string `json:"file"`
}

type OpenCodeFileEditedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeFileEdited struct {
	Data     OpenCodeFileEditedData     `json:"data"`
	Durable  *OpenCodeFileEditedDurable `json:"durable,omitempty"`
	ID       string                     `json:"id"`
	Location OpenCodeLocationRef        `json:"location,omitempty"`
	Metadata map[string]any             `json:"metadata,omitempty"`
	Type     string                     `json:"type"`
}

type OpenCodeFileNode struct {
	Absolute string `json:"absolute"`
	Ignored  bool   `json:"ignored"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"`
}

type OpenCodeFilePart struct {
	Filename  string                 `json:"filename,omitempty"`
	ID        string                 `json:"id"`
	MessageID string                 `json:"messageID"`
	Mime      string                 `json:"mime"`
	SessionID string                 `json:"sessionID"`
	Source    OpenCodeFilePartSource `json:"source,omitempty"`
	Type      string                 `json:"type"`
	Url       string                 `json:"url"`
}

type OpenCodeFilePartInput struct {
	Filename string                 `json:"filename,omitempty"`
	ID       string                 `json:"id,omitempty"`
	Mime     string                 `json:"mime"`
	Source   OpenCodeFilePartSource `json:"source,omitempty"`
	Type     string                 `json:"type"`
	Url      string                 `json:"url"`
}

type OpenCodeFilePartSource any

type OpenCodeFilePartSourceText struct {
	End   float64 `json:"end"`
	Start float64 `json:"start"`
	Value string  `json:"value"`
}

type OpenCodeFileSource struct {
	Path string                     `json:"path"`
	Text OpenCodeFilePartSourceText `json:"text"`
	Type string                     `json:"type"`
}

type OpenCodeFileSystemEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type OpenCodeFileWatcherUpdatedData struct {
	Event string `json:"event"`
	File  string `json:"file"`
}

type OpenCodeFileWatcherUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeFileWatcherUpdated struct {
	Data     OpenCodeFileWatcherUpdatedData     `json:"data"`
	Durable  *OpenCodeFileWatcherUpdatedDurable `json:"durable,omitempty"`
	ID       string                             `json:"id"`
	Location OpenCodeLocationRef                `json:"location,omitempty"`
	Metadata map[string]any                     `json:"metadata,omitempty"`
	Type     string                             `json:"type"`
}

type OpenCodeForbiddenError struct {
	Tag     string `json:"_tag"`
	Message string `json:"message"`
}

type OpenCodeFormatterStatus struct {
	Enabled    bool     `json:"enabled"`
	Extensions []string `json:"extensions"`
	Name       string   `json:"name"`
}

type OpenCodeGlobalDisposedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeGlobalDisposed struct {
	Data     map[string]any                 `json:"data"`
	Durable  *OpenCodeGlobalDisposedDurable `json:"durable,omitempty"`
	ID       string                         `json:"id"`
	Location OpenCodeLocationRef            `json:"location,omitempty"`
	Metadata map[string]any                 `json:"metadata,omitempty"`
	Type     string                         `json:"type"`
}

type OpenCodeGlobalEvent struct {
	Directory string `json:"directory"`
	Payload   any    `json:"payload"`
	Project   string `json:"project,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}

type OpenCodeGlobalSessionModel struct {
	ID         string `json:"id"`
	ProviderID string `json:"providerID"`
	Variant    string `json:"variant,omitempty"`
}

type OpenCodeGlobalSessionRevert struct {
	Diff      string `json:"diff,omitempty"`
	MessageID string `json:"messageID"`
	PartID    string `json:"partID,omitempty"`
	Snapshot  string `json:"snapshot,omitempty"`
}

type OpenCodeGlobalSessionShare struct {
	Url string `json:"url"`
}

type OpenCodeGlobalSessionSummary struct {
	Additions float64                    `json:"additions"`
	Deletions float64                    `json:"deletions"`
	Diffs     []OpenCodeSnapshotFileDiff `json:"diffs,omitempty"`
	Files     float64                    `json:"files"`
}

type OpenCodeGlobalSessionTime struct {
	Archived   float64 `json:"archived,omitempty"`
	Compacting int64   `json:"compacting,omitempty"`
	Created    int64   `json:"created"`
	Updated    int64   `json:"updated"`
}

type OpenCodeGlobalSessionTokensCache struct {
	Read  float64 `json:"read"`
	Write float64 `json:"write"`
}

type OpenCodeGlobalSessionTokens struct {
	Cache     OpenCodeGlobalSessionTokensCache `json:"cache"`
	Input     float64                          `json:"input"`
	Output    float64                          `json:"output"`
	Reasoning float64                          `json:"reasoning"`
}

type OpenCodeGlobalSession struct {
	Agent       string                        `json:"agent,omitempty"`
	Cost        float64                       `json:"cost,omitempty"`
	Directory   string                        `json:"directory"`
	ID          string                        `json:"id"`
	Metadata    map[string]any                `json:"metadata,omitempty"`
	Model       *OpenCodeGlobalSessionModel   `json:"model,omitempty"`
	ParentID    string                        `json:"parentID,omitempty"`
	Path        string                        `json:"path,omitempty"`
	Permission  OpenCodePermissionRuleset     `json:"permission,omitempty"`
	Project     any                           `json:"project"`
	ProjectID   string                        `json:"projectID"`
	Revert      *OpenCodeGlobalSessionRevert  `json:"revert,omitempty"`
	Share       *OpenCodeGlobalSessionShare   `json:"share,omitempty"`
	Slug        string                        `json:"slug"`
	Summary     *OpenCodeGlobalSessionSummary `json:"summary,omitempty"`
	Time        OpenCodeGlobalSessionTime     `json:"time"`
	Title       string                        `json:"title"`
	Tokens      *OpenCodeGlobalSessionTokens  `json:"tokens,omitempty"`
	Version     string                        `json:"version"`
	WorkspaceID string                        `json:"workspaceID,omitempty"`
}

type OpenCodeImageAttachmentConfig struct {
	AutoResize     bool  `json:"auto_resize,omitempty"`
	MaxBase64Bytes int64 `json:"max_base64_bytes,omitempty"`
	MaxHeight      int64 `json:"max_height,omitempty"`
	MaxWidth       int64 `json:"max_width,omitempty"`
}

type OpenCodeInstallationUpdateAvailableData struct {
	Version string `json:"version"`
}

type OpenCodeInstallationUpdateAvailableDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeInstallationUpdateAvailable struct {
	Data     OpenCodeInstallationUpdateAvailableData     `json:"data"`
	Durable  *OpenCodeInstallationUpdateAvailableDurable `json:"durable,omitempty"`
	ID       string                                      `json:"id"`
	Location OpenCodeLocationRef                         `json:"location,omitempty"`
	Metadata map[string]any                              `json:"metadata,omitempty"`
	Type     string                                      `json:"type"`
}

type OpenCodeInstallationUpdatedData struct {
	Version string `json:"version"`
}

type OpenCodeInstallationUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeInstallationUpdated struct {
	Data     OpenCodeInstallationUpdatedData     `json:"data"`
	Durable  *OpenCodeInstallationUpdatedDurable `json:"durable,omitempty"`
	ID       string                              `json:"id"`
	Location OpenCodeLocationRef                 `json:"location,omitempty"`
	Metadata map[string]any                      `json:"metadata,omitempty"`
	Type     string                              `json:"type"`
}

type OpenCodeIntegrationAttemptTime struct {
	Created any `json:"created"`
	Expires any `json:"expires"`
}

type OpenCodeIntegrationAttempt struct {
	AttemptID    string                         `json:"attemptID"`
	Instructions string                         `json:"instructions"`
	Mode         string                         `json:"mode"`
	Time         OpenCodeIntegrationAttemptTime `json:"time"`
	Url          string                         `json:"url"`
}

type OpenCodeIntegrationAttemptStatus any

type OpenCodeIntegrationConnectionUpdatedData struct {
	IntegrationID string `json:"integrationID"`
}

type OpenCodeIntegrationConnectionUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeIntegrationConnectionUpdated struct {
	Data     OpenCodeIntegrationConnectionUpdatedData     `json:"data"`
	Durable  *OpenCodeIntegrationConnectionUpdatedDurable `json:"durable,omitempty"`
	ID       string                                       `json:"id"`
	Location OpenCodeLocationRef                          `json:"location,omitempty"`
	Metadata map[string]any                               `json:"metadata,omitempty"`
	Type     string                                       `json:"type"`
}

type OpenCodeIntegrationEnvMethod struct {
	Names []string `json:"names"`
	Type  string   `json:"type"`
}

type OpenCodeIntegrationInfo struct {
	Connections []OpenCodeConnectionInfo    `json:"connections"`
	ID          string                      `json:"id"`
	Methods     []OpenCodeIntegrationMethod `json:"methods"`
	Name        string                      `json:"name"`
}

type OpenCodeIntegrationInputs struct {
}

type OpenCodeIntegrationKeyMethod struct {
	Label string `json:"label,omitempty"`
	Type  string `json:"type"`
}

type OpenCodeIntegrationMethod any

type OpenCodeIntegrationOAuthMethod struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Prompts []any  `json:"prompts,omitempty"`
	Type    string `json:"type"`
}

type OpenCodeIntegrationRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type OpenCodeIntegrationSelectPromptOptionsItem struct {
	Hint  string `json:"hint,omitempty"`
	Label string `json:"label"`
	Value string `json:"value"`
}

type OpenCodeIntegrationSelectPrompt struct {
	Key     string                                       `json:"key"`
	Message string                                       `json:"message"`
	Options []OpenCodeIntegrationSelectPromptOptionsItem `json:"options"`
	Type    string                                       `json:"type"`
	When    OpenCodeIntegrationWhen                      `json:"when,omitempty"`
}

type OpenCodeIntegrationTextPrompt struct {
	Key         string                  `json:"key"`
	Message     string                  `json:"message"`
	Placeholder string                  `json:"placeholder,omitempty"`
	Type        string                  `json:"type"`
	When        OpenCodeIntegrationWhen `json:"when,omitempty"`
}

type OpenCodeIntegrationUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeIntegrationUpdated struct {
	Data     map[string]any                     `json:"data"`
	Durable  *OpenCodeIntegrationUpdatedDurable `json:"durable,omitempty"`
	ID       string                             `json:"id"`
	Location OpenCodeLocationRef                `json:"location,omitempty"`
	Metadata map[string]any                     `json:"metadata,omitempty"`
	Type     string                             `json:"type"`
}

type OpenCodeIntegrationWhen struct {
	Key   string `json:"key"`
	Op    string `json:"op"`
	Value string `json:"value"`
}

type OpenCodeInvalidCursorError struct {
	Tag     string `json:"_tag"`
	Message string `json:"message"`
}

type OpenCodeInvalidRequestError struct {
	Tag     string `json:"_tag"`
	Field   string `json:"field,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Message string `json:"message"`
}

type OpenCodeJSONSchema struct {
}

type OpenCodeLLMProviderMetadata struct {
}

type OpenCodeLLMToolContent any

type OpenCodeLSPStatus struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Root   string `json:"root"`
	Status string `json:"status"`
}

type OpenCodeLayoutConfig string

type OpenCodeLocationInfoProject struct {
	Directory string `json:"directory"`
	ID        string `json:"id"`
}

type OpenCodeLocationInfo struct {
	Directory   string                      `json:"directory"`
	Project     OpenCodeLocationInfoProject `json:"project"`
	WorkspaceID string                      `json:"workspaceID,omitempty"`
}

type OpenCodeLocationRef struct {
	Directory   string `json:"directory"`
	WorkspaceID string `json:"workspaceID,omitempty"`
}

type OpenCodeLogLevel string

type OpenCodeLspUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeLspUpdated struct {
	Data     map[string]any             `json:"data"`
	Durable  *OpenCodeLspUpdatedDurable `json:"durable,omitempty"`
	ID       string                     `json:"id"`
	Location OpenCodeLocationRef        `json:"location,omitempty"`
	Metadata map[string]any             `json:"metadata,omitempty"`
	Type     string                     `json:"type"`
}

type OpenCodeMCPStatus any

type OpenCodeMCPStatusConnected struct {
	Status string `json:"status"`
}

type OpenCodeMCPStatusDisabled struct {
	Status string `json:"status"`
}

type OpenCodeMCPStatusFailed struct {
	Error  string `json:"error"`
	Status string `json:"status"`
}

type OpenCodeMCPStatusNeedsAuth struct {
	Status string `json:"status"`
}

type OpenCodeMCPStatusNeedsClientRegistration struct {
	Error  string `json:"error"`
	Status string `json:"status"`
}

type OpenCodeMcpBrowserOpenFailedData struct {
	McpName string `json:"mcpName"`
	Url     string `json:"url"`
}

type OpenCodeMcpBrowserOpenFailedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeMcpBrowserOpenFailed struct {
	Data     OpenCodeMcpBrowserOpenFailedData     `json:"data"`
	Durable  *OpenCodeMcpBrowserOpenFailedDurable `json:"durable,omitempty"`
	ID       string                               `json:"id"`
	Location OpenCodeLocationRef                  `json:"location,omitempty"`
	Metadata map[string]any                       `json:"metadata,omitempty"`
	Type     string                               `json:"type"`
}

type OpenCodeMcpLocalConfig struct {
	Command     []string          `json:"command"`
	Cwd         string            `json:"cwd,omitempty"`
	Enabled     bool              `json:"enabled,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Timeout     int64             `json:"timeout,omitempty"`
	Type        string            `json:"type"`
}

type OpenCodeMcpOAuthConfig struct {
	CallbackPort int64  `json:"callbackPort,omitempty"`
	ClientId     string `json:"clientId,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
	RedirectUri  string `json:"redirectUri,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type OpenCodeMcpRemoteConfig struct {
	Enabled bool              `json:"enabled,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Oauth   any               `json:"oauth,omitempty"`
	Timeout int64             `json:"timeout,omitempty"`
	Type    string            `json:"type"`
	Url     string            `json:"url"`
}

type OpenCodeMcpResource struct {
	Client      string `json:"client"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	Name        string `json:"name"`
	Uri         string `json:"uri"`
}

type OpenCodeMcpServerNotFoundError struct {
	Tag     string `json:"_tag"`
	Message string `json:"message"`
	Name    string `json:"name"`
}

type OpenCodeMcpToolsChangedData struct {
	Server string `json:"server"`
}

type OpenCodeMcpToolsChangedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeMcpToolsChanged struct {
	Data     OpenCodeMcpToolsChangedData     `json:"data"`
	Durable  *OpenCodeMcpToolsChangedDurable `json:"durable,omitempty"`
	ID       string                          `json:"id"`
	Location OpenCodeLocationRef             `json:"location,omitempty"`
	Metadata map[string]any                  `json:"metadata,omitempty"`
	Type     string                          `json:"type"`
}

type OpenCodeMcpUnsupportedOAuthError struct {
	Error string `json:"error"`
}

type OpenCodeMessage any

type OpenCodeMessageAbortedErrorData struct {
	Message string `json:"message"`
}

type OpenCodeMessageAbortedError struct {
	Data OpenCodeMessageAbortedErrorData `json:"data"`
	Name string                          `json:"name"`
}

type OpenCodeMessageNotFoundError struct {
	Tag       string `json:"_tag"`
	Message   string `json:"message"`
	MessageID string `json:"messageID"`
	SessionID string `json:"sessionID"`
}

type OpenCodeMessageOutputLengthError struct {
	Data map[string]any `json:"data"`
	Name string         `json:"name"`
}

type OpenCodeMessagePartDeltaData struct {
	Delta     string `json:"delta"`
	Field     string `json:"field"`
	MessageID string `json:"messageID"`
	PartID    string `json:"partID"`
	SessionID string `json:"sessionID"`
}

type OpenCodeMessagePartDeltaDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeMessagePartDelta struct {
	Data     OpenCodeMessagePartDeltaData     `json:"data"`
	Durable  *OpenCodeMessagePartDeltaDurable `json:"durable,omitempty"`
	ID       string                           `json:"id"`
	Location OpenCodeLocationRef              `json:"location,omitempty"`
	Metadata map[string]any                   `json:"metadata,omitempty"`
	Type     string                           `json:"type"`
}

type OpenCodeMessagePartRemovedData struct {
	MessageID string `json:"messageID"`
	PartID    string `json:"partID"`
	SessionID string `json:"sessionID"`
}

type OpenCodeMessagePartRemovedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeMessagePartRemoved struct {
	Data     OpenCodeMessagePartRemovedData     `json:"data"`
	Durable  *OpenCodeMessagePartRemovedDurable `json:"durable,omitempty"`
	ID       string                             `json:"id"`
	Location OpenCodeLocationRef                `json:"location,omitempty"`
	Metadata map[string]any                     `json:"metadata,omitempty"`
	Type     string                             `json:"type"`
}

type OpenCodeMessagePartUpdatedData struct {
	Part      OpenCodePart `json:"part"`
	SessionID string       `json:"sessionID"`
	Time      float64      `json:"time"`
}

type OpenCodeMessagePartUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeMessagePartUpdated struct {
	Data     OpenCodeMessagePartUpdatedData     `json:"data"`
	Durable  *OpenCodeMessagePartUpdatedDurable `json:"durable,omitempty"`
	ID       string                             `json:"id"`
	Location OpenCodeLocationRef                `json:"location,omitempty"`
	Metadata map[string]any                     `json:"metadata,omitempty"`
	Type     string                             `json:"type"`
}

type OpenCodeMessageRemovedData struct {
	MessageID string `json:"messageID"`
	SessionID string `json:"sessionID"`
}

type OpenCodeMessageRemovedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeMessageRemoved struct {
	Data     OpenCodeMessageRemovedData     `json:"data"`
	Durable  *OpenCodeMessageRemovedDurable `json:"durable,omitempty"`
	ID       string                         `json:"id"`
	Location OpenCodeLocationRef            `json:"location,omitempty"`
	Metadata map[string]any                 `json:"metadata,omitempty"`
	Type     string                         `json:"type"`
}

type OpenCodeMessageUpdatedData struct {
	Info      OpenCodeMessage `json:"info"`
	SessionID string          `json:"sessionID"`
}

type OpenCodeMessageUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeMessageUpdated struct {
	Data     OpenCodeMessageUpdatedData     `json:"data"`
	Durable  *OpenCodeMessageUpdatedDurable `json:"durable,omitempty"`
	ID       string                         `json:"id"`
	Location OpenCodeLocationRef            `json:"location,omitempty"`
	Metadata map[string]any                 `json:"metadata,omitempty"`
	Type     string                         `json:"type"`
}

type OpenCodeModelApi2 struct {
	ID  string `json:"id"`
	Npm string `json:"npm"`
	Url string `json:"url"`
}

type OpenCodeModelCapabilities2Input struct {
	Audio bool `json:"audio"`
	Image bool `json:"image"`
	Pdf   bool `json:"pdf"`
	Text  bool `json:"text"`
	Video bool `json:"video"`
}

type OpenCodeModelCapabilities2Output struct {
	Audio bool `json:"audio"`
	Image bool `json:"image"`
	Pdf   bool `json:"pdf"`
	Text  bool `json:"text"`
	Video bool `json:"video"`
}

type OpenCodeModelCapabilities2 struct {
	Attachment  bool                             `json:"attachment"`
	Input       OpenCodeModelCapabilities2Input  `json:"input"`
	Interleaved any                              `json:"interleaved"`
	Output      OpenCodeModelCapabilities2Output `json:"output"`
	Reasoning   bool                             `json:"reasoning"`
	Temperature bool                             `json:"temperature"`
	Toolcall    bool                             `json:"toolcall"`
}

type OpenCodeModelCost2Cache struct {
	Read  float64 `json:"read"`
	Write float64 `json:"write"`
}

type OpenCodeModelCost2ExperimentalOver200KCache struct {
	Read  float64 `json:"read"`
	Write float64 `json:"write"`
}

type OpenCodeModelCost2ExperimentalOver200K struct {
	Cache  OpenCodeModelCost2ExperimentalOver200KCache `json:"cache"`
	Input  float64                                     `json:"input"`
	Output float64                                     `json:"output"`
}

type OpenCodeModelCost2TiersItemCache struct {
	Read  float64 `json:"read"`
	Write float64 `json:"write"`
}

type OpenCodeModelCost2TiersItemTier struct {
	Size float64 `json:"size"`
	Type string  `json:"type"`
}

type OpenCodeModelCost2TiersItem struct {
	Cache  OpenCodeModelCost2TiersItemCache `json:"cache"`
	Input  float64                          `json:"input"`
	Output float64                          `json:"output"`
	Tier   OpenCodeModelCost2TiersItemTier  `json:"tier"`
}

type OpenCodeModelCost2 struct {
	Cache                OpenCodeModelCost2Cache                 `json:"cache"`
	ExperimentalOver200K *OpenCodeModelCost2ExperimentalOver200K `json:"experimentalOver200K,omitempty"`
	Input                float64                                 `json:"input"`
	Output               float64                                 `json:"output"`
	Tiers                []OpenCodeModelCost2TiersItem           `json:"tiers,omitempty"`
}

type OpenCodeModelLimit struct {
	Context float64 `json:"context"`
	Input   float64 `json:"input,omitempty"`
	Output  float64 `json:"output"`
}

type OpenCodeModel struct {
	Api          OpenCodeModelApi2          `json:"api"`
	Capabilities OpenCodeModelCapabilities2 `json:"capabilities"`
	Cost         OpenCodeModelCost2         `json:"cost"`
	Family       string                     `json:"family,omitempty"`
	Headers      map[string]string          `json:"headers"`
	ID           string                     `json:"id"`
	Limit        OpenCodeModelLimit         `json:"limit"`
	Name         string                     `json:"name"`
	Options      map[string]any             `json:"options"`
	ProviderID   string                     `json:"providerID"`
	ReleaseDate  string                     `json:"release_date"`
	Status       string                     `json:"status"`
	Variants     map[string]map[string]any  `json:"variants,omitempty"`
}

type OpenCodeModelApi any

type OpenCodeModelCapabilities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
	Tools  bool     `json:"tools"`
}

type OpenCodeModelCostCache struct {
	Read  float64 `json:"read"`
	Write float64 `json:"write"`
}

type OpenCodeModelCostTier struct {
	Size int64  `json:"size"`
	Type string `json:"type"`
}

type OpenCodeModelCost struct {
	Cache  OpenCodeModelCostCache `json:"cache"`
	Input  float64                `json:"input"`
	Output float64                `json:"output"`
	Tier   *OpenCodeModelCostTier `json:"tier,omitempty"`
}

type OpenCodeModelRef struct {
	ID         string `json:"id"`
	ProviderID string `json:"providerID"`
	Variant    string `json:"variant,omitempty"`
}

type OpenCodeModelV2InfoLimit struct {
	Context int64 `json:"context"`
	Input   int64 `json:"input,omitempty"`
	Output  int64 `json:"output"`
}

type OpenCodeModelV2InfoRequest struct {
	Body    map[string]any    `json:"body"`
	Headers map[string]string `json:"headers"`
	Variant string            `json:"variant,omitempty"`
}

type OpenCodeModelV2InfoTime struct {
	Released float64 `json:"released"`
}

type OpenCodeModelV2InfoVariantsItem struct {
	Body    map[string]any    `json:"body"`
	Headers map[string]string `json:"headers"`
	ID      string            `json:"id"`
}

type OpenCodeModelV2Info struct {
	Api          OpenCodeModelApi                  `json:"api"`
	Capabilities OpenCodeModelCapabilities         `json:"capabilities"`
	Cost         []OpenCodeModelCost               `json:"cost"`
	Enabled      bool                              `json:"enabled"`
	Family       string                            `json:"family,omitempty"`
	ID           string                            `json:"id"`
	Limit        OpenCodeModelV2InfoLimit          `json:"limit"`
	Name         string                            `json:"name"`
	ProviderID   string                            `json:"providerID"`
	Request      OpenCodeModelV2InfoRequest        `json:"request"`
	Status       string                            `json:"status"`
	Time         OpenCodeModelV2InfoTime           `json:"time"`
	Variants     []OpenCodeModelV2InfoVariantsItem `json:"variants"`
}

type OpenCodeModelsDevRefreshedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeModelsDevRefreshed struct {
	Data     map[string]any                     `json:"data"`
	Durable  *OpenCodeModelsDevRefreshedDurable `json:"durable,omitempty"`
	ID       string                             `json:"id"`
	Location OpenCodeLocationRef                `json:"location,omitempty"`
	Metadata map[string]any                     `json:"metadata,omitempty"`
	Type     string                             `json:"type"`
}

type OpenCodeMoveSessionDestination struct {
	Directory string `json:"directory"`
}

type OpenCodeMoveSessionErrorData struct {
	Message string `json:"message"`
}

type OpenCodeMoveSessionError struct {
	Data OpenCodeMoveSessionErrorData `json:"data"`
	Name string                       `json:"name"`
}

type OpenCodeNotFoundErrorData struct {
	Message string `json:"message"`
}

type OpenCodeNotFoundError struct {
	Data OpenCodeNotFoundErrorData `json:"data"`
	Name string                    `json:"name"`
}

type OpenCodeOAuth struct {
	Access        string `json:"access"`
	AccountId     string `json:"accountId,omitempty"`
	EnterpriseUrl string `json:"enterpriseUrl,omitempty"`
	Expires       int64  `json:"expires"`
	Refresh       string `json:"refresh"`
	Type          string `json:"type"`
}

type OpenCodeOutputFormat any

type OpenCodeOutputFormat1 any

type OpenCodeOutputFormatJsonSchema struct {
	RetryCount int64              `json:"retryCount,omitempty"`
	Schema     OpenCodeJSONSchema `json:"schema"`
	Type       string             `json:"type"`
}

type OpenCodeOutputFormatText struct {
	Type string `json:"type"`
}

type OpenCodePart any

type OpenCodePatchPart struct {
	Files     []string `json:"files"`
	Hash      string   `json:"hash"`
	ID        string   `json:"id"`
	MessageID string   `json:"messageID"`
	SessionID string   `json:"sessionID"`
	Type      string   `json:"type"`
}

type OpenCodePath struct {
	Config    string `json:"config"`
	Directory string `json:"directory"`
	Home      string `json:"home"`
	State     string `json:"state"`
	Worktree  string `json:"worktree"`
}

type OpenCodePermissionAction string

type OpenCodePermissionActionConfig string

type OpenCodePermissionAskedDataTool struct {
	CallID    string `json:"callID"`
	MessageID string `json:"messageID"`
}

type OpenCodePermissionAskedData struct {
	Always     []string                         `json:"always"`
	ID         string                           `json:"id"`
	Metadata   map[string]any                   `json:"metadata"`
	Patterns   []string                         `json:"patterns"`
	Permission string                           `json:"permission"`
	SessionID  string                           `json:"sessionID"`
	Tool       *OpenCodePermissionAskedDataTool `json:"tool,omitempty"`
}

type OpenCodePermissionAskedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodePermissionAsked struct {
	Data     OpenCodePermissionAskedData     `json:"data"`
	Durable  *OpenCodePermissionAskedDurable `json:"durable,omitempty"`
	ID       string                          `json:"id"`
	Location OpenCodeLocationRef             `json:"location,omitempty"`
	Metadata map[string]any                  `json:"metadata,omitempty"`
	Type     string                          `json:"type"`
}

type OpenCodePermissionConfig any

type OpenCodePermissionNotFoundError struct {
	Tag       string `json:"_tag"`
	Message   string `json:"message"`
	RequestID string `json:"requestID"`
}

type OpenCodePermissionObjectConfig struct {
}

type OpenCodePermissionRepliedData struct {
	Reply     string `json:"reply"`
	RequestID string `json:"requestID"`
	SessionID string `json:"sessionID"`
}

type OpenCodePermissionRepliedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodePermissionReplied struct {
	Data     OpenCodePermissionRepliedData     `json:"data"`
	Durable  *OpenCodePermissionRepliedDurable `json:"durable,omitempty"`
	ID       string                            `json:"id"`
	Location OpenCodeLocationRef               `json:"location,omitempty"`
	Metadata map[string]any                    `json:"metadata,omitempty"`
	Type     string                            `json:"type"`
}

type OpenCodePermissionRequestTool struct {
	CallID    string `json:"callID"`
	MessageID string `json:"messageID"`
}

type OpenCodePermissionRequest struct {
	Always     []string                       `json:"always"`
	ID         string                         `json:"id"`
	Metadata   map[string]any                 `json:"metadata"`
	Patterns   []string                       `json:"patterns"`
	Permission string                         `json:"permission"`
	SessionID  string                         `json:"sessionID"`
	Tool       *OpenCodePermissionRequestTool `json:"tool,omitempty"`
}

type OpenCodePermissionRule struct {
	Action     OpenCodePermissionAction `json:"action"`
	Pattern    string                   `json:"pattern"`
	Permission string                   `json:"permission"`
}

type OpenCodePermissionRuleConfig any

type OpenCodePermissionRuleset []OpenCodePermissionRule

type OpenCodePermissionSavedInfo struct {
	Action    string `json:"action"`
	ID        string `json:"id"`
	ProjectID string `json:"projectID"`
	Resource  string `json:"resource"`
}

type OpenCodePermissionV2AskedData struct {
	Action    string                     `json:"action"`
	ID        string                     `json:"id"`
	Metadata  map[string]any             `json:"metadata,omitempty"`
	Resources []string                   `json:"resources"`
	Save      []string                   `json:"save,omitempty"`
	SessionID string                     `json:"sessionID"`
	Source    OpenCodePermissionV2Source `json:"source,omitempty"`
}

type OpenCodePermissionV2AskedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodePermissionV2Asked struct {
	Data     OpenCodePermissionV2AskedData     `json:"data"`
	Durable  *OpenCodePermissionV2AskedDurable `json:"durable,omitempty"`
	ID       string                            `json:"id"`
	Location OpenCodeLocationRef               `json:"location,omitempty"`
	Metadata map[string]any                    `json:"metadata,omitempty"`
	Type     string                            `json:"type"`
}

type OpenCodePermissionV2Effect string

type OpenCodePermissionV2RepliedData struct {
	Reply     OpenCodePermissionV2Reply `json:"reply"`
	RequestID string                    `json:"requestID"`
	SessionID string                    `json:"sessionID"`
}

type OpenCodePermissionV2RepliedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodePermissionV2Replied struct {
	Data     OpenCodePermissionV2RepliedData     `json:"data"`
	Durable  *OpenCodePermissionV2RepliedDurable `json:"durable,omitempty"`
	ID       string                              `json:"id"`
	Location OpenCodeLocationRef                 `json:"location,omitempty"`
	Metadata map[string]any                      `json:"metadata,omitempty"`
	Type     string                              `json:"type"`
}

type OpenCodePermissionV2Reply string

type OpenCodePermissionV2Request struct {
	Action    string                     `json:"action"`
	ID        string                     `json:"id"`
	Metadata  map[string]any             `json:"metadata,omitempty"`
	Resources []string                   `json:"resources"`
	Save      []string                   `json:"save,omitempty"`
	SessionID string                     `json:"sessionID"`
	Source    OpenCodePermissionV2Source `json:"source,omitempty"`
}

type OpenCodePermissionV2Rule struct {
	Action   string                     `json:"action"`
	Effect   OpenCodePermissionV2Effect `json:"effect"`
	Resource string                     `json:"resource"`
}

type OpenCodePermissionV2Ruleset []OpenCodePermissionV2Rule

type OpenCodePermissionV2Source struct {
	CallID    string `json:"callID"`
	MessageID string `json:"messageID"`
	Type      string `json:"type"`
}

type OpenCodePluginAddedData struct {
	ID string `json:"id"`
}

type OpenCodePluginAddedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodePluginAdded struct {
	Data     OpenCodePluginAddedData     `json:"data"`
	Durable  *OpenCodePluginAddedDurable `json:"durable,omitempty"`
	ID       string                      `json:"id"`
	Location OpenCodeLocationRef         `json:"location,omitempty"`
	Metadata map[string]any              `json:"metadata,omitempty"`
	Type     string                      `json:"type"`
}

type OpenCodePolicyEffect string

type OpenCodeProject struct {
	Commands  OpenCodeProjectCommands `json:"commands,omitempty"`
	Icon      OpenCodeProjectIcon     `json:"icon,omitempty"`
	ID        string                  `json:"id"`
	Name      string                  `json:"name,omitempty"`
	Sandboxes []string                `json:"sandboxes"`
	Time      OpenCodeProjectTime     `json:"time"`
	Vcs       OpenCodeProjectVcs      `json:"vcs,omitempty"`
	Worktree  string                  `json:"worktree"`
}

type OpenCodeProjectCommands struct {
	Start string `json:"start,omitempty"`
}

type OpenCodeProjectCopyCopy struct {
	Directory string `json:"directory"`
}

type OpenCodeProjectCopyErrorData struct {
	ForceRequired bool   `json:"forceRequired,omitempty"`
	Message       string `json:"message"`
}

type OpenCodeProjectCopyError struct {
	Data OpenCodeProjectCopyErrorData `json:"data"`
	Name string                       `json:"name"`
}

type OpenCodeProjectDirectories []struct {
	Directory string `json:"directory"`
	Strategy  string `json:"strategy,omitempty"`
}

type OpenCodeProjectDirectoriesUpdatedData struct {
	ProjectID string `json:"projectID"`
}

type OpenCodeProjectDirectoriesUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeProjectDirectoriesUpdated struct {
	Data     OpenCodeProjectDirectoriesUpdatedData     `json:"data"`
	Durable  *OpenCodeProjectDirectoriesUpdatedDurable `json:"durable,omitempty"`
	ID       string                                    `json:"id"`
	Location OpenCodeLocationRef                       `json:"location,omitempty"`
	Metadata map[string]any                            `json:"metadata,omitempty"`
	Type     string                                    `json:"type"`
}

type OpenCodeProjectIcon struct {
	Color    string `json:"color,omitempty"`
	Override string `json:"override,omitempty"`
	Url      string `json:"url,omitempty"`
}

type OpenCodeProjectNotFoundError struct {
	Tag       string `json:"_tag"`
	Message   string `json:"message"`
	ProjectID string `json:"projectID"`
}

type OpenCodeProjectSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Worktree string `json:"worktree"`
}

type OpenCodeProjectTime struct {
	Created     int64 `json:"created"`
	Initialized int64 `json:"initialized,omitempty"`
	Updated     int64 `json:"updated"`
}

type OpenCodeProjectUpdatedData struct {
	Commands  OpenCodeProjectCommands `json:"commands,omitempty"`
	Icon      OpenCodeProjectIcon     `json:"icon,omitempty"`
	ID        string                  `json:"id"`
	Name      string                  `json:"name,omitempty"`
	Sandboxes []string                `json:"sandboxes"`
	Time      OpenCodeProjectTime     `json:"time"`
	Vcs       OpenCodeProjectVcs      `json:"vcs,omitempty"`
	Worktree  string                  `json:"worktree"`
}

type OpenCodeProjectUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeProjectUpdated struct {
	Data     OpenCodeProjectUpdatedData     `json:"data"`
	Durable  *OpenCodeProjectUpdatedDurable `json:"durable,omitempty"`
	ID       string                         `json:"id"`
	Location OpenCodeLocationRef            `json:"location,omitempty"`
	Metadata map[string]any                 `json:"metadata,omitempty"`
	Type     string                         `json:"type"`
}

type OpenCodeProjectVcs string

type OpenCodePrompt struct {
	Agents []OpenCodePromptAgentAttachment `json:"agents,omitempty"`
	Files  []OpenCodePromptFileAttachment  `json:"files,omitempty"`
	Text   string                          `json:"text"`
}

type OpenCodePromptAgentAttachment struct {
	Name   string               `json:"name"`
	Source OpenCodePromptSource `json:"source,omitempty"`
}

type OpenCodePromptFileAttachment struct {
	Description string               `json:"description,omitempty"`
	Mime        string               `json:"mime"`
	Name        string               `json:"name,omitempty"`
	Source      OpenCodePromptSource `json:"source,omitempty"`
	Uri         string               `json:"uri"`
}

type OpenCodePromptInput struct {
	Agents []OpenCodePromptAgentAttachment     `json:"agents,omitempty"`
	Files  []OpenCodePromptInputFileAttachment `json:"files,omitempty"`
	Text   string                              `json:"text"`
}

type OpenCodePromptInputFileAttachment struct {
	Description string               `json:"description,omitempty"`
	Name        string               `json:"name,omitempty"`
	Source      OpenCodePromptSource `json:"source,omitempty"`
	Uri         string               `json:"uri"`
}

type OpenCodePromptSource struct {
	End   float64 `json:"end"`
	Start float64 `json:"start"`
	Text  string  `json:"text"`
}

type OpenCodeProvider struct {
	Env     []string                 `json:"env"`
	ID      string                   `json:"id"`
	Key     string                   `json:"key,omitempty"`
	Models  map[string]OpenCodeModel `json:"models"`
	Name    string                   `json:"name"`
	Options map[string]any           `json:"options"`
	Source  string                   `json:"source"`
}

type OpenCodeProviderAISDK struct {
	Package  string         `json:"package"`
	Settings map[string]any `json:"settings,omitempty"`
	Type     string         `json:"type"`
	Url      string         `json:"url,omitempty"`
}

type OpenCodeProviderApi any

type OpenCodeProviderAuthAuthorization struct {
	Instructions string `json:"instructions"`
	Method       string `json:"method"`
	Url          string `json:"url"`
}

type OpenCodeProviderAuthErrorData struct {
	Message    string `json:"message"`
	ProviderID string `json:"providerID"`
}

type OpenCodeProviderAuthError struct {
	Data OpenCodeProviderAuthErrorData `json:"data"`
	Name string                        `json:"name"`
}

type OpenCodeProviderAuthError1Data struct {
	Field      string `json:"field,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Message    string `json:"message,omitempty"`
	ProviderID string `json:"providerID,omitempty"`
}

type OpenCodeProviderAuthError1 struct {
	Data OpenCodeProviderAuthError1Data `json:"data"`
	Name string                         `json:"name"`
}

type OpenCodeProviderAuthMethod struct {
	Label   string `json:"label"`
	Prompts []any  `json:"prompts,omitempty"`
	Type    string `json:"type"`
}

type OpenCodeProviderConfigOptions struct {
	ApiKey        string `json:"apiKey,omitempty"`
	BaseURL       string `json:"baseURL,omitempty"`
	ChunkTimeout  int64  `json:"chunkTimeout,omitempty"`
	EnterpriseUrl string `json:"enterpriseUrl,omitempty"`
	HeaderTimeout any    `json:"headerTimeout,omitempty"`
	SetCacheKey   bool   `json:"setCacheKey,omitempty"`
	Timeout       any    `json:"timeout,omitempty"`
}

type OpenCodeProviderConfig struct {
	Api       string   `json:"api,omitempty"`
	Blacklist []string `json:"blacklist,omitempty"`
	Env       []string `json:"env,omitempty"`
	ID        string   `json:"id,omitempty"`
	Models    map[string]struct {
		Attachment bool `json:"attachment,omitempty"`
		Cost       struct {
			CacheRead       float64 `json:"cache_read,omitempty"`
			CacheWrite      float64 `json:"cache_write,omitempty"`
			ContextOver200k struct {
				CacheRead  float64 `json:"cache_read,omitempty"`
				CacheWrite float64 `json:"cache_write,omitempty"`
				Input      float64 `json:"input"`
				Output     float64 `json:"output"`
			} `json:"context_over_200k,omitempty"`
			Input  float64 `json:"input"`
			Output float64 `json:"output"`
		} `json:"cost,omitempty"`
		Experimental bool              `json:"experimental,omitempty"`
		Family       string            `json:"family,omitempty"`
		Headers      map[string]string `json:"headers,omitempty"`
		ID           string            `json:"id,omitempty"`
		Interleaved  any               `json:"interleaved,omitempty"`
		Limit        struct {
			Context float64 `json:"context"`
			Input   float64 `json:"input,omitempty"`
			Output  float64 `json:"output"`
		} `json:"limit,omitempty"`
		Modalities struct {
			Input  []string `json:"input,omitempty"`
			Output []string `json:"output,omitempty"`
		} `json:"modalities,omitempty"`
		Name     string         `json:"name,omitempty"`
		Options  map[string]any `json:"options,omitempty"`
		Provider struct {
			Api string `json:"api,omitempty"`
			Npm string `json:"npm,omitempty"`
		} `json:"provider,omitempty"`
		Reasoning   bool   `json:"reasoning,omitempty"`
		ReleaseDate string `json:"release_date,omitempty"`
		Status      string `json:"status,omitempty"`
		Temperature bool   `json:"temperature,omitempty"`
		ToolCall    bool   `json:"tool_call,omitempty"`
		Variants    map[string]struct {
			Disabled bool `json:"disabled,omitempty"`
		} `json:"variants,omitempty"`
	} `json:"models,omitempty"`
	Name      string                         `json:"name,omitempty"`
	Npm       string                         `json:"npm,omitempty"`
	Options   *OpenCodeProviderConfigOptions `json:"options,omitempty"`
	Whitelist []string                       `json:"whitelist,omitempty"`
}

type OpenCodeProviderNative struct {
	Settings map[string]any `json:"settings"`
	Type     string         `json:"type"`
	Url      string         `json:"url,omitempty"`
}

type OpenCodeProviderNotFoundError struct {
	Tag        string `json:"_tag"`
	Message    string `json:"message"`
	ProviderID string `json:"providerID"`
}

type OpenCodeProviderRequest struct {
	Body    map[string]any    `json:"body"`
	Headers map[string]string `json:"headers"`
}

type OpenCodeProviderV2Info struct {
	Api           OpenCodeProviderApi     `json:"api"`
	Disabled      bool                    `json:"disabled,omitempty"`
	ID            string                  `json:"id"`
	IntegrationID string                  `json:"integrationID,omitempty"`
	Name          string                  `json:"name"`
	Request       OpenCodeProviderRequest `json:"request"`
}

type OpenCodePty struct {
	Args     []string `json:"args"`
	Command  string   `json:"command"`
	Cwd      string   `json:"cwd"`
	ExitCode int64    `json:"exitCode,omitempty"`
	ID       string   `json:"id"`
	Pid      int64    `json:"pid"`
	Status   string   `json:"status"`
	Title    string   `json:"title"`
}

type OpenCodePtyCreatedData struct {
	Info OpenCodePty `json:"info"`
}

type OpenCodePtyCreatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodePtyCreated struct {
	Data     OpenCodePtyCreatedData     `json:"data"`
	Durable  *OpenCodePtyCreatedDurable `json:"durable,omitempty"`
	ID       string                     `json:"id"`
	Location OpenCodeLocationRef        `json:"location,omitempty"`
	Metadata map[string]any             `json:"metadata,omitempty"`
	Type     string                     `json:"type"`
}

type OpenCodePtyDeletedData struct {
	ID string `json:"id"`
}

type OpenCodePtyDeletedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodePtyDeleted struct {
	Data     OpenCodePtyDeletedData     `json:"data"`
	Durable  *OpenCodePtyDeletedDurable `json:"durable,omitempty"`
	ID       string                     `json:"id"`
	Location OpenCodeLocationRef        `json:"location,omitempty"`
	Metadata map[string]any             `json:"metadata,omitempty"`
	Type     string                     `json:"type"`
}

type OpenCodePtyExitedData struct {
	ExitCode int64  `json:"exitCode"`
	ID       string `json:"id"`
}

type OpenCodePtyExitedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodePtyExited struct {
	Data     OpenCodePtyExitedData     `json:"data"`
	Durable  *OpenCodePtyExitedDurable `json:"durable,omitempty"`
	ID       string                    `json:"id"`
	Location OpenCodeLocationRef       `json:"location,omitempty"`
	Metadata map[string]any            `json:"metadata,omitempty"`
	Type     string                    `json:"type"`
}

type OpenCodePtyForbiddenError struct {
	Tag     string `json:"_tag"`
	Message string `json:"message"`
}

type OpenCodePtyNotFoundError struct {
	Tag     string `json:"_tag"`
	Message string `json:"message"`
	PtyID   string `json:"ptyID"`
}

type OpenCodePtyTicketConnectToken struct {
	ExpiresIn int64  `json:"expires_in"`
	Ticket    string `json:"ticket"`
}

type OpenCodePtyUpdatedData struct {
	Info OpenCodePty `json:"info"`
}

type OpenCodePtyUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodePtyUpdated struct {
	Data     OpenCodePtyUpdatedData     `json:"data"`
	Durable  *OpenCodePtyUpdatedDurable `json:"durable,omitempty"`
	ID       string                     `json:"id"`
	Location OpenCodeLocationRef        `json:"location,omitempty"`
	Metadata map[string]any             `json:"metadata,omitempty"`
	Type     string                     `json:"type"`
}

type OpenCodeQuestionAnswer []string

type OpenCodeQuestionAskedData struct {
	ID        string                 `json:"id"`
	Questions []OpenCodeQuestionInfo `json:"questions"`
	SessionID string                 `json:"sessionID"`
	Tool      OpenCodeQuestionTool   `json:"tool,omitempty"`
}

type OpenCodeQuestionAskedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeQuestionAsked struct {
	Data     OpenCodeQuestionAskedData     `json:"data"`
	Durable  *OpenCodeQuestionAskedDurable `json:"durable,omitempty"`
	ID       string                        `json:"id"`
	Location OpenCodeLocationRef           `json:"location,omitempty"`
	Metadata map[string]any                `json:"metadata,omitempty"`
	Type     string                        `json:"type"`
}

type OpenCodeQuestionInfo struct {
	Custom   bool                     `json:"custom,omitempty"`
	Header   string                   `json:"header"`
	Multiple bool                     `json:"multiple,omitempty"`
	Options  []OpenCodeQuestionOption `json:"options"`
	Question string                   `json:"question"`
}

type OpenCodeQuestionNotFoundError struct {
	Tag       string `json:"_tag"`
	Message   string `json:"message"`
	RequestID string `json:"requestID"`
}

type OpenCodeQuestionOption struct {
	Description string `json:"description"`
	Label       string `json:"label"`
}

type OpenCodeQuestionRejected struct {
	RequestID string `json:"requestID"`
	SessionID string `json:"sessionID"`
}

type OpenCodeQuestionReplied struct {
	Answers   []OpenCodeQuestionAnswer `json:"answers"`
	RequestID string                   `json:"requestID"`
	SessionID string                   `json:"sessionID"`
}

type OpenCodeQuestionRequest struct {
	ID        string                 `json:"id"`
	Questions []OpenCodeQuestionInfo `json:"questions"`
	SessionID string                 `json:"sessionID"`
	Tool      OpenCodeQuestionTool   `json:"tool,omitempty"`
}

type OpenCodeQuestionTool struct {
	CallID    string `json:"callID"`
	MessageID string `json:"messageID"`
}

type OpenCodeQuestionV2Answer []string

type OpenCodeQuestionV2AskedData struct {
	ID        string                   `json:"id"`
	Questions []OpenCodeQuestionV2Info `json:"questions"`
	SessionID string                   `json:"sessionID"`
	Tool      OpenCodeQuestionV2Tool   `json:"tool,omitempty"`
}

type OpenCodeQuestionV2AskedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeQuestionV2Asked struct {
	Data     OpenCodeQuestionV2AskedData     `json:"data"`
	Durable  *OpenCodeQuestionV2AskedDurable `json:"durable,omitempty"`
	ID       string                          `json:"id"`
	Location OpenCodeLocationRef             `json:"location,omitempty"`
	Metadata map[string]any                  `json:"metadata,omitempty"`
	Type     string                          `json:"type"`
}

type OpenCodeQuestionV2Info struct {
	Custom   bool                       `json:"custom,omitempty"`
	Header   string                     `json:"header"`
	Multiple bool                       `json:"multiple,omitempty"`
	Options  []OpenCodeQuestionV2Option `json:"options"`
	Question string                     `json:"question"`
}

type OpenCodeQuestionV2Option struct {
	Description string `json:"description"`
	Label       string `json:"label"`
}

type OpenCodeQuestionV2RejectedData struct {
	RequestID string `json:"requestID"`
	SessionID string `json:"sessionID"`
}

type OpenCodeQuestionV2RejectedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeQuestionV2Rejected struct {
	Data     OpenCodeQuestionV2RejectedData     `json:"data"`
	Durable  *OpenCodeQuestionV2RejectedDurable `json:"durable,omitempty"`
	ID       string                             `json:"id"`
	Location OpenCodeLocationRef                `json:"location,omitempty"`
	Metadata map[string]any                     `json:"metadata,omitempty"`
	Type     string                             `json:"type"`
}

type OpenCodeQuestionV2RepliedData struct {
	Answers   []OpenCodeQuestionV2Answer `json:"answers"`
	RequestID string                     `json:"requestID"`
	SessionID string                     `json:"sessionID"`
}

type OpenCodeQuestionV2RepliedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeQuestionV2Replied struct {
	Data     OpenCodeQuestionV2RepliedData     `json:"data"`
	Durable  *OpenCodeQuestionV2RepliedDurable `json:"durable,omitempty"`
	ID       string                            `json:"id"`
	Location OpenCodeLocationRef               `json:"location,omitempty"`
	Metadata map[string]any                    `json:"metadata,omitempty"`
	Type     string                            `json:"type"`
}

type OpenCodeQuestionV2Reply struct {
	Answers []OpenCodeQuestionV2Answer `json:"answers"`
}

type OpenCodeQuestionV2Request struct {
	ID        string                   `json:"id"`
	Questions []OpenCodeQuestionV2Info `json:"questions"`
	SessionID string                   `json:"sessionID"`
	Tool      OpenCodeQuestionV2Tool   `json:"tool,omitempty"`
}

type OpenCodeQuestionV2Tool struct {
	CallID    string `json:"callID"`
	MessageID string `json:"messageID"`
}

type OpenCodeRangeEnd struct {
	Character int64 `json:"character"`
	Line      int64 `json:"line"`
}

type OpenCodeRangeStart struct {
	Character int64 `json:"character"`
	Line      int64 `json:"line"`
}

type OpenCodeRange struct {
	End   OpenCodeRangeEnd   `json:"end"`
	Start OpenCodeRangeStart `json:"start"`
}

type OpenCodeReasoningPartTime struct {
	End   int64 `json:"end,omitempty"`
	Start int64 `json:"start"`
}

type OpenCodeReasoningPart struct {
	ID        string                    `json:"id"`
	MessageID string                    `json:"messageID"`
	Metadata  map[string]any            `json:"metadata,omitempty"`
	SessionID string                    `json:"sessionID"`
	Text      string                    `json:"text"`
	Time      OpenCodeReasoningPartTime `json:"time"`
	Type      string                    `json:"type"`
}

type OpenCodeReferenceGitSource struct {
	Branch      string `json:"branch,omitempty"`
	Description string `json:"description,omitempty"`
	Hidden      bool   `json:"hidden,omitempty"`
	Repository  string `json:"repository"`
	Type        string `json:"type"`
}

type OpenCodeReferenceInfo struct {
	Description string                  `json:"description,omitempty"`
	Hidden      bool                    `json:"hidden,omitempty"`
	Name        string                  `json:"name"`
	Path        string                  `json:"path"`
	Source      OpenCodeReferenceSource `json:"source"`
}

type OpenCodeReferenceLocalSource struct {
	Description string `json:"description,omitempty"`
	Hidden      bool   `json:"hidden,omitempty"`
	Path        string `json:"path"`
	Type        string `json:"type"`
}

type OpenCodeReferenceSource any

type OpenCodeReferenceUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeReferenceUpdated struct {
	Data     map[string]any                   `json:"data"`
	Durable  *OpenCodeReferenceUpdatedDurable `json:"durable,omitempty"`
	ID       string                           `json:"id"`
	Location OpenCodeLocationRef              `json:"location,omitempty"`
	Metadata map[string]any                   `json:"metadata,omitempty"`
	Type     string                           `json:"type"`
}

type OpenCodeResourceSource struct {
	ClientName string                     `json:"clientName"`
	Text       OpenCodeFilePartSourceText `json:"text"`
	Type       string                     `json:"type"`
	Uri        string                     `json:"uri"`
}

type OpenCodeRetryPartTime struct {
	Created int64 `json:"created"`
}

type OpenCodeRetryPart struct {
	Attempt   int64                 `json:"attempt"`
	Error     OpenCodeAPIError      `json:"error"`
	ID        string                `json:"id"`
	MessageID string                `json:"messageID"`
	SessionID string                `json:"sessionID"`
	Time      OpenCodeRetryPartTime `json:"time"`
	Type      string                `json:"type"`
}

type OpenCodeRevertState struct {
	Diff      string             `json:"diff,omitempty"`
	Files     []OpenCodeFileDiff `json:"files,omitempty"`
	MessageID string             `json:"messageID"`
	PartID    string             `json:"partID,omitempty"`
	Snapshot  string             `json:"snapshot,omitempty"`
}

type OpenCodeServerConfig struct {
	Cors       []string `json:"cors,omitempty"`
	Hostname   string   `json:"hostname,omitempty"`
	Mdns       bool     `json:"mdns,omitempty"`
	MdnsDomain string   `json:"mdnsDomain,omitempty"`
	Port       int64    `json:"port,omitempty"`
}

type OpenCodeServerConnectedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeServerConnected struct {
	Data     map[string]any                  `json:"data"`
	Durable  *OpenCodeServerConnectedDurable `json:"durable,omitempty"`
	ID       string                          `json:"id"`
	Location OpenCodeLocationRef             `json:"location,omitempty"`
	Metadata map[string]any                  `json:"metadata,omitempty"`
	Type     string                          `json:"type"`
}

type OpenCodeServiceUnavailableError struct {
	Tag     string `json:"_tag"`
	Message string `json:"message"`
	Service string `json:"service,omitempty"`
}

type OpenCodeSessionModel struct {
	ID         string `json:"id"`
	ProviderID string `json:"providerID"`
	Variant    string `json:"variant,omitempty"`
}

type OpenCodeSessionRevert struct {
	Diff      string `json:"diff,omitempty"`
	MessageID string `json:"messageID"`
	PartID    string `json:"partID,omitempty"`
	Snapshot  string `json:"snapshot,omitempty"`
}

type OpenCodeSessionShare struct {
	Url string `json:"url"`
}

type OpenCodeSessionSummary struct {
	Additions float64                    `json:"additions"`
	Deletions float64                    `json:"deletions"`
	Diffs     []OpenCodeSnapshotFileDiff `json:"diffs,omitempty"`
	Files     float64                    `json:"files"`
}

type OpenCodeSessionTime struct {
	Archived   float64 `json:"archived,omitempty"`
	Compacting int64   `json:"compacting,omitempty"`
	Created    int64   `json:"created"`
	Updated    int64   `json:"updated"`
}

type OpenCodeSessionTokensCache struct {
	Read  float64 `json:"read"`
	Write float64 `json:"write"`
}

type OpenCodeSessionTokens struct {
	Cache     OpenCodeSessionTokensCache `json:"cache"`
	Input     float64                    `json:"input"`
	Output    float64                    `json:"output"`
	Reasoning float64                    `json:"reasoning"`
}

type OpenCodeSession struct {
	Agent       string                    `json:"agent,omitempty"`
	Cost        float64                   `json:"cost,omitempty"`
	Directory   string                    `json:"directory"`
	ID          string                    `json:"id"`
	Metadata    map[string]any            `json:"metadata,omitempty"`
	Model       *OpenCodeSessionModel     `json:"model,omitempty"`
	ParentID    string                    `json:"parentID,omitempty"`
	Path        string                    `json:"path,omitempty"`
	Permission  OpenCodePermissionRuleset `json:"permission,omitempty"`
	ProjectID   string                    `json:"projectID"`
	Revert      *OpenCodeSessionRevert    `json:"revert,omitempty"`
	Share       *OpenCodeSessionShare     `json:"share,omitempty"`
	Slug        string                    `json:"slug"`
	Summary     *OpenCodeSessionSummary   `json:"summary,omitempty"`
	Time        OpenCodeSessionTime       `json:"time"`
	Title       string                    `json:"title"`
	Tokens      *OpenCodeSessionTokens    `json:"tokens,omitempty"`
	Version     string                    `json:"version"`
	WorkspaceID string                    `json:"workspaceID,omitempty"`
}

type OpenCodeSessionActive struct {
	Type string `json:"type"`
}

type OpenCodeSessionBusyError struct {
	Tag       string `json:"_tag"`
	Message   string `json:"message"`
	SessionID string `json:"sessionID"`
}

type OpenCodeSessionCompactedData struct {
	SessionID string `json:"sessionID"`
}

type OpenCodeSessionCompactedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionCompacted struct {
	Data     OpenCodeSessionCompactedData     `json:"data"`
	Durable  *OpenCodeSessionCompactedDurable `json:"durable,omitempty"`
	ID       string                           `json:"id"`
	Location OpenCodeLocationRef              `json:"location,omitempty"`
	Metadata map[string]any                   `json:"metadata,omitempty"`
	Type     string                           `json:"type"`
}

type OpenCodeSessionCreatedData struct {
	Info      OpenCodeSession `json:"info"`
	SessionID string          `json:"sessionID"`
}

type OpenCodeSessionCreatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionCreated struct {
	Data     OpenCodeSessionCreatedData     `json:"data"`
	Durable  *OpenCodeSessionCreatedDurable `json:"durable,omitempty"`
	ID       string                         `json:"id"`
	Location OpenCodeLocationRef            `json:"location,omitempty"`
	Metadata map[string]any                 `json:"metadata,omitempty"`
	Type     string                         `json:"type"`
}

type OpenCodeSessionDeletedData struct {
	Info      OpenCodeSession `json:"info"`
	SessionID string          `json:"sessionID"`
}

type OpenCodeSessionDeletedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionDeleted struct {
	Data     OpenCodeSessionDeletedData     `json:"data"`
	Durable  *OpenCodeSessionDeletedDurable `json:"durable,omitempty"`
	ID       string                         `json:"id"`
	Location OpenCodeLocationRef            `json:"location,omitempty"`
	Metadata map[string]any                 `json:"metadata,omitempty"`
	Type     string                         `json:"type"`
}

type OpenCodeSessionDiffData struct {
	Diff      []OpenCodeSnapshotFileDiff `json:"diff"`
	SessionID string                     `json:"sessionID"`
}

type OpenCodeSessionDiffDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionDiff struct {
	Data     OpenCodeSessionDiffData     `json:"data"`
	Durable  *OpenCodeSessionDiffDurable `json:"durable,omitempty"`
	ID       string                      `json:"id"`
	Location OpenCodeLocationRef         `json:"location,omitempty"`
	Metadata map[string]any              `json:"metadata,omitempty"`
	Type     string                      `json:"type"`
}

type OpenCodeSessionDurableEvent any

type OpenCodeSessionDurableEventStream string

type OpenCodeSessionErrorData struct {
	Error     any    `json:"error,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
}

type OpenCodeSessionErrorDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionError struct {
	Data     OpenCodeSessionErrorData     `json:"data"`
	Durable  *OpenCodeSessionErrorDurable `json:"durable,omitempty"`
	ID       string                       `json:"id"`
	Location OpenCodeLocationRef          `json:"location,omitempty"`
	Metadata map[string]any               `json:"metadata,omitempty"`
	Type     string                       `json:"type"`
}

type OpenCodeSessionErrorUnknown struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type OpenCodeSessionHistory struct {
	Data    []OpenCodeSessionDurableEvent `json:"data"`
	HasMore bool                          `json:"hasMore"`
}

type OpenCodeSessionIdleData struct {
	SessionID string `json:"sessionID"`
}

type OpenCodeSessionIdleDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionIdle struct {
	Data     OpenCodeSessionIdleData     `json:"data"`
	Durable  *OpenCodeSessionIdleDurable `json:"durable,omitempty"`
	ID       string                      `json:"id"`
	Location OpenCodeLocationRef         `json:"location,omitempty"`
	Metadata map[string]any              `json:"metadata,omitempty"`
	Type     string                      `json:"type"`
}

type OpenCodeSessionInputAdmitted struct {
	AdmittedSeq int64          `json:"admittedSeq"`
	Delivery    string         `json:"delivery"`
	ID          string         `json:"id"`
	PromotedSeq int64          `json:"promotedSeq,omitempty"`
	Prompt      OpenCodePrompt `json:"prompt"`
	SessionID   string         `json:"sessionID"`
	TimeCreated float64        `json:"timeCreated"`
}

type OpenCodeSessionMessage any

type OpenCodeSessionMessageAgentSwitchedTime struct {
	Created float64 `json:"created"`
}

type OpenCodeSessionMessageAgentSwitched struct {
	Agent    string                                  `json:"agent"`
	ID       string                                  `json:"id"`
	Metadata map[string]any                          `json:"metadata,omitempty"`
	Time     OpenCodeSessionMessageAgentSwitchedTime `json:"time"`
	Type     string                                  `json:"type"`
}

type OpenCodeSessionMessageAssistantSnapshot struct {
	End   string   `json:"end,omitempty"`
	Files []string `json:"files,omitempty"`
	Start string   `json:"start,omitempty"`
}

type OpenCodeSessionMessageAssistantTime struct {
	Completed float64 `json:"completed,omitempty"`
	Created   float64 `json:"created"`
}

type OpenCodeSessionMessageAssistantTokensCache struct {
	Read  float64 `json:"read"`
	Write float64 `json:"write"`
}

type OpenCodeSessionMessageAssistantTokens struct {
	Cache     OpenCodeSessionMessageAssistantTokensCache `json:"cache"`
	Input     float64                                    `json:"input"`
	Output    float64                                    `json:"output"`
	Reasoning float64                                    `json:"reasoning"`
}

type OpenCodeSessionMessageAssistant struct {
	Agent    string                                   `json:"agent"`
	Content  []any                                    `json:"content"`
	Cost     float64                                  `json:"cost,omitempty"`
	Error    OpenCodeSessionErrorUnknown              `json:"error,omitempty"`
	Finish   string                                   `json:"finish,omitempty"`
	ID       string                                   `json:"id"`
	Metadata map[string]any                           `json:"metadata,omitempty"`
	Model    OpenCodeModelRef                         `json:"model"`
	Snapshot *OpenCodeSessionMessageAssistantSnapshot `json:"snapshot,omitempty"`
	Time     OpenCodeSessionMessageAssistantTime      `json:"time"`
	Tokens   *OpenCodeSessionMessageAssistantTokens   `json:"tokens,omitempty"`
	Type     string                                   `json:"type"`
}

type OpenCodeSessionMessageAssistantReasoningTime struct {
	Completed float64 `json:"completed,omitempty"`
	Created   float64 `json:"created"`
}

type OpenCodeSessionMessageAssistantReasoning struct {
	ID               string                                        `json:"id"`
	ProviderMetadata OpenCodeLLMProviderMetadata                   `json:"providerMetadata,omitempty"`
	Text             string                                        `json:"text"`
	Time             *OpenCodeSessionMessageAssistantReasoningTime `json:"time,omitempty"`
	Type             string                                        `json:"type"`
}

type OpenCodeSessionMessageAssistantText struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Type string `json:"type"`
}

type OpenCodeSessionMessageAssistantToolProvider struct {
	Executed       bool                        `json:"executed"`
	Metadata       OpenCodeLLMProviderMetadata `json:"metadata,omitempty"`
	ResultMetadata OpenCodeLLMProviderMetadata `json:"resultMetadata,omitempty"`
}

type OpenCodeSessionMessageAssistantToolTime struct {
	Completed float64 `json:"completed,omitempty"`
	Created   float64 `json:"created"`
	Pruned    float64 `json:"pruned,omitempty"`
	Ran       float64 `json:"ran,omitempty"`
}

type OpenCodeSessionMessageAssistantTool struct {
	ID       string                                       `json:"id"`
	Name     string                                       `json:"name"`
	Provider *OpenCodeSessionMessageAssistantToolProvider `json:"provider,omitempty"`
	State    any                                          `json:"state"`
	Time     OpenCodeSessionMessageAssistantToolTime      `json:"time"`
	Type     string                                       `json:"type"`
}

type OpenCodeSessionMessageCompactionTime struct {
	Created float64 `json:"created"`
}

type OpenCodeSessionMessageCompaction struct {
	ID       string                               `json:"id"`
	Metadata map[string]any                       `json:"metadata,omitempty"`
	Reason   string                               `json:"reason"`
	Recent   string                               `json:"recent"`
	Summary  string                               `json:"summary"`
	Time     OpenCodeSessionMessageCompactionTime `json:"time"`
	Type     string                               `json:"type"`
}

type OpenCodeSessionMessageModelSwitchedTime struct {
	Created float64 `json:"created"`
}

type OpenCodeSessionMessageModelSwitched struct {
	ID       string                                  `json:"id"`
	Metadata map[string]any                          `json:"metadata,omitempty"`
	Model    OpenCodeModelRef                        `json:"model"`
	Time     OpenCodeSessionMessageModelSwitchedTime `json:"time"`
	Type     string                                  `json:"type"`
}

type OpenCodeSessionMessageShellTime struct {
	Completed float64 `json:"completed,omitempty"`
	Created   float64 `json:"created"`
}

type OpenCodeSessionMessageShell struct {
	CallID   string                          `json:"callID"`
	Command  string                          `json:"command"`
	ID       string                          `json:"id"`
	Metadata map[string]any                  `json:"metadata,omitempty"`
	Output   string                          `json:"output"`
	Time     OpenCodeSessionMessageShellTime `json:"time"`
	Type     string                          `json:"type"`
}

type OpenCodeSessionMessageSyntheticTime struct {
	Created float64 `json:"created"`
}

type OpenCodeSessionMessageSynthetic struct {
	ID        string                              `json:"id"`
	Metadata  map[string]any                      `json:"metadata,omitempty"`
	SessionID string                              `json:"sessionID"`
	Text      string                              `json:"text"`
	Time      OpenCodeSessionMessageSyntheticTime `json:"time"`
	Type      string                              `json:"type"`
}

type OpenCodeSessionMessageSystemTime struct {
	Created float64 `json:"created"`
}

type OpenCodeSessionMessageSystem struct {
	ID       string                           `json:"id"`
	Metadata map[string]any                   `json:"metadata,omitempty"`
	Text     string                           `json:"text"`
	Time     OpenCodeSessionMessageSystemTime `json:"time"`
	Type     string                           `json:"type"`
}

type OpenCodeSessionMessageToolStateCompleted struct {
	Attachments []OpenCodePromptFileAttachment `json:"attachments,omitempty"`
	Content     []OpenCodeLLMToolContent       `json:"content"`
	Input       map[string]any                 `json:"input"`
	OutputPaths []string                       `json:"outputPaths,omitempty"`
	Result      any                            `json:"result,omitempty"`
	Status      string                         `json:"status"`
	Structured  map[string]any                 `json:"structured"`
}

type OpenCodeSessionMessageToolStateError struct {
	Content    []OpenCodeLLMToolContent    `json:"content"`
	Error      OpenCodeSessionErrorUnknown `json:"error"`
	Input      map[string]any              `json:"input"`
	Result     any                         `json:"result,omitempty"`
	Status     string                      `json:"status"`
	Structured map[string]any              `json:"structured"`
}

type OpenCodeSessionMessageToolStatePending struct {
	Input  string `json:"input"`
	Status string `json:"status"`
}

type OpenCodeSessionMessageToolStateRunning struct {
	Content    []OpenCodeLLMToolContent `json:"content"`
	Input      map[string]any           `json:"input"`
	Status     string                   `json:"status"`
	Structured map[string]any           `json:"structured"`
}

type OpenCodeSessionMessageUserTime struct {
	Created float64 `json:"created"`
}

type OpenCodeSessionMessageUser struct {
	Agents   []OpenCodePromptAgentAttachment `json:"agents,omitempty"`
	Files    []OpenCodePromptFileAttachment  `json:"files,omitempty"`
	ID       string                          `json:"id"`
	Metadata map[string]any                  `json:"metadata,omitempty"`
	Text     string                          `json:"text"`
	Time     OpenCodeSessionMessageUserTime  `json:"time"`
	Type     string                          `json:"type"`
}

type OpenCodeSessionMessagesResponseCursor struct {
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
}

type OpenCodeSessionMessagesResponse struct {
	Cursor OpenCodeSessionMessagesResponseCursor `json:"cursor"`
	Data   []OpenCodeSessionMessage              `json:"data"`
}

type OpenCodeSessionNextAgentSwitchedData struct {
	Agent     string  `json:"agent"`
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSessionNextAgentSwitchedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextAgentSwitched struct {
	Data     OpenCodeSessionNextAgentSwitchedData     `json:"data"`
	Durable  *OpenCodeSessionNextAgentSwitchedDurable `json:"durable,omitempty"`
	ID       string                                   `json:"id"`
	Location OpenCodeLocationRef                      `json:"location,omitempty"`
	Metadata map[string]any                           `json:"metadata,omitempty"`
	Type     string                                   `json:"type"`
}

type OpenCodeSessionNextCompactionDeltaData struct {
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSessionNextCompactionDeltaDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextCompactionDelta struct {
	Data     OpenCodeSessionNextCompactionDeltaData     `json:"data"`
	Durable  *OpenCodeSessionNextCompactionDeltaDurable `json:"durable,omitempty"`
	ID       string                                     `json:"id"`
	Location OpenCodeLocationRef                        `json:"location,omitempty"`
	Metadata map[string]any                             `json:"metadata,omitempty"`
	Type     string                                     `json:"type"`
}

type OpenCodeSessionNextCompactionEndedData struct {
	MessageID string  `json:"messageID"`
	Reason    string  `json:"reason"`
	Recent    string  `json:"recent"`
	SessionID string  `json:"sessionID"`
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSessionNextCompactionEndedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextCompactionEnded struct {
	Data     OpenCodeSessionNextCompactionEndedData     `json:"data"`
	Durable  *OpenCodeSessionNextCompactionEndedDurable `json:"durable,omitempty"`
	ID       string                                     `json:"id"`
	Location OpenCodeLocationRef                        `json:"location,omitempty"`
	Metadata map[string]any                             `json:"metadata,omitempty"`
	Type     string                                     `json:"type"`
}

type OpenCodeSessionNextCompactionStartedData struct {
	MessageID string  `json:"messageID"`
	Reason    string  `json:"reason"`
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSessionNextCompactionStartedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextCompactionStarted struct {
	Data     OpenCodeSessionNextCompactionStartedData     `json:"data"`
	Durable  *OpenCodeSessionNextCompactionStartedDurable `json:"durable,omitempty"`
	ID       string                                       `json:"id"`
	Location OpenCodeLocationRef                          `json:"location,omitempty"`
	Metadata map[string]any                               `json:"metadata,omitempty"`
	Type     string                                       `json:"type"`
}

type OpenCodeSessionNextContextUpdatedData struct {
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSessionNextContextUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextContextUpdated struct {
	Data     OpenCodeSessionNextContextUpdatedData     `json:"data"`
	Durable  *OpenCodeSessionNextContextUpdatedDurable `json:"durable,omitempty"`
	ID       string                                    `json:"id"`
	Location OpenCodeLocationRef                       `json:"location,omitempty"`
	Metadata map[string]any                            `json:"metadata,omitempty"`
	Type     string                                    `json:"type"`
}

type OpenCodeSessionNextModelSwitchedData struct {
	MessageID string           `json:"messageID"`
	Model     OpenCodeModelRef `json:"model"`
	SessionID string           `json:"sessionID"`
	Timestamp float64          `json:"timestamp"`
}

type OpenCodeSessionNextModelSwitchedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextModelSwitched struct {
	Data     OpenCodeSessionNextModelSwitchedData     `json:"data"`
	Durable  *OpenCodeSessionNextModelSwitchedDurable `json:"durable,omitempty"`
	ID       string                                   `json:"id"`
	Location OpenCodeLocationRef                      `json:"location,omitempty"`
	Metadata map[string]any                           `json:"metadata,omitempty"`
	Type     string                                   `json:"type"`
}

type OpenCodeSessionNextMovedData struct {
	Location     OpenCodeLocationRef `json:"location"`
	SessionID    string              `json:"sessionID"`
	Subdirectory string              `json:"subdirectory,omitempty"`
	Timestamp    float64             `json:"timestamp"`
}

type OpenCodeSessionNextMovedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextMoved struct {
	Data     OpenCodeSessionNextMovedData     `json:"data"`
	Durable  *OpenCodeSessionNextMovedDurable `json:"durable,omitempty"`
	ID       string                           `json:"id"`
	Location OpenCodeLocationRef              `json:"location,omitempty"`
	Metadata map[string]any                   `json:"metadata,omitempty"`
	Type     string                           `json:"type"`
}

type OpenCodeSessionNextPromptAdmittedData struct {
	Delivery  string         `json:"delivery"`
	MessageID string         `json:"messageID"`
	Prompt    OpenCodePrompt `json:"prompt"`
	SessionID string         `json:"sessionID"`
	Timestamp float64        `json:"timestamp"`
}

type OpenCodeSessionNextPromptAdmittedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextPromptAdmitted struct {
	Data     OpenCodeSessionNextPromptAdmittedData     `json:"data"`
	Durable  *OpenCodeSessionNextPromptAdmittedDurable `json:"durable,omitempty"`
	ID       string                                    `json:"id"`
	Location OpenCodeLocationRef                       `json:"location,omitempty"`
	Metadata map[string]any                            `json:"metadata,omitempty"`
	Type     string                                    `json:"type"`
}

type OpenCodeSessionNextPromptedData struct {
	Delivery  string         `json:"delivery"`
	MessageID string         `json:"messageID"`
	Prompt    OpenCodePrompt `json:"prompt"`
	SessionID string         `json:"sessionID"`
	Timestamp float64        `json:"timestamp"`
}

type OpenCodeSessionNextPromptedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextPrompted struct {
	Data     OpenCodeSessionNextPromptedData     `json:"data"`
	Durable  *OpenCodeSessionNextPromptedDurable `json:"durable,omitempty"`
	ID       string                              `json:"id"`
	Location OpenCodeLocationRef                 `json:"location,omitempty"`
	Metadata map[string]any                      `json:"metadata,omitempty"`
	Type     string                              `json:"type"`
}

type OpenCodeSessionNextReasoningDeltaData struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	Delta              string  `json:"delta"`
	ReasoningID        string  `json:"reasoningID"`
	SessionID          string  `json:"sessionID"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeSessionNextReasoningDeltaDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextReasoningDelta struct {
	Data     OpenCodeSessionNextReasoningDeltaData     `json:"data"`
	Durable  *OpenCodeSessionNextReasoningDeltaDurable `json:"durable,omitempty"`
	ID       string                                    `json:"id"`
	Location OpenCodeLocationRef                       `json:"location,omitempty"`
	Metadata map[string]any                            `json:"metadata,omitempty"`
	Type     string                                    `json:"type"`
}

type OpenCodeSessionNextReasoningEndedData struct {
	AssistantMessageID string                      `json:"assistantMessageID"`
	ProviderMetadata   OpenCodeLLMProviderMetadata `json:"providerMetadata,omitempty"`
	ReasoningID        string                      `json:"reasoningID"`
	SessionID          string                      `json:"sessionID"`
	Text               string                      `json:"text"`
	Timestamp          float64                     `json:"timestamp"`
}

type OpenCodeSessionNextReasoningEndedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextReasoningEnded struct {
	Data     OpenCodeSessionNextReasoningEndedData     `json:"data"`
	Durable  *OpenCodeSessionNextReasoningEndedDurable `json:"durable,omitempty"`
	ID       string                                    `json:"id"`
	Location OpenCodeLocationRef                       `json:"location,omitempty"`
	Metadata map[string]any                            `json:"metadata,omitempty"`
	Type     string                                    `json:"type"`
}

type OpenCodeSessionNextReasoningStartedData struct {
	AssistantMessageID string                      `json:"assistantMessageID"`
	ProviderMetadata   OpenCodeLLMProviderMetadata `json:"providerMetadata,omitempty"`
	ReasoningID        string                      `json:"reasoningID"`
	SessionID          string                      `json:"sessionID"`
	Timestamp          float64                     `json:"timestamp"`
}

type OpenCodeSessionNextReasoningStartedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextReasoningStarted struct {
	Data     OpenCodeSessionNextReasoningStartedData     `json:"data"`
	Durable  *OpenCodeSessionNextReasoningStartedDurable `json:"durable,omitempty"`
	ID       string                                      `json:"id"`
	Location OpenCodeLocationRef                         `json:"location,omitempty"`
	Metadata map[string]any                              `json:"metadata,omitempty"`
	Type     string                                      `json:"type"`
}

type OpenCodeSessionNextRetriedData struct {
	Attempt   float64                       `json:"attempt"`
	Error     OpenCodeSessionNextRetryError `json:"error"`
	SessionID string                        `json:"sessionID"`
	Timestamp float64                       `json:"timestamp"`
}

type OpenCodeSessionNextRetriedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextRetried struct {
	Data     OpenCodeSessionNextRetriedData     `json:"data"`
	Durable  *OpenCodeSessionNextRetriedDurable `json:"durable,omitempty"`
	ID       string                             `json:"id"`
	Location OpenCodeLocationRef                `json:"location,omitempty"`
	Metadata map[string]any                     `json:"metadata,omitempty"`
	Type     string                             `json:"type"`
}

type OpenCodeSessionNextRetryError struct {
	IsRetryable     bool              `json:"isRetryable"`
	Message         string            `json:"message"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	ResponseBody    string            `json:"responseBody,omitempty"`
	ResponseHeaders map[string]string `json:"responseHeaders,omitempty"`
	StatusCode      float64           `json:"statusCode,omitempty"`
}

type OpenCodeSessionNextRevertClearedData struct {
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSessionNextRevertClearedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextRevertCleared struct {
	Data     OpenCodeSessionNextRevertClearedData     `json:"data"`
	Durable  *OpenCodeSessionNextRevertClearedDurable `json:"durable,omitempty"`
	ID       string                                   `json:"id"`
	Location OpenCodeLocationRef                      `json:"location,omitempty"`
	Metadata map[string]any                           `json:"metadata,omitempty"`
	Type     string                                   `json:"type"`
}

type OpenCodeSessionNextRevertCommittedData struct {
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSessionNextRevertCommittedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextRevertCommitted struct {
	Data     OpenCodeSessionNextRevertCommittedData     `json:"data"`
	Durable  *OpenCodeSessionNextRevertCommittedDurable `json:"durable,omitempty"`
	ID       string                                     `json:"id"`
	Location OpenCodeLocationRef                        `json:"location,omitempty"`
	Metadata map[string]any                             `json:"metadata,omitempty"`
	Type     string                                     `json:"type"`
}

type OpenCodeSessionNextRevertStagedData struct {
	Revert    OpenCodeRevertState `json:"revert"`
	SessionID string              `json:"sessionID"`
	Timestamp float64             `json:"timestamp"`
}

type OpenCodeSessionNextRevertStagedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextRevertStaged struct {
	Data     OpenCodeSessionNextRevertStagedData     `json:"data"`
	Durable  *OpenCodeSessionNextRevertStagedDurable `json:"durable,omitempty"`
	ID       string                                  `json:"id"`
	Location OpenCodeLocationRef                     `json:"location,omitempty"`
	Metadata map[string]any                          `json:"metadata,omitempty"`
	Type     string                                  `json:"type"`
}

type OpenCodeSessionNextShellEndedData struct {
	CallID    string  `json:"callID"`
	Output    string  `json:"output"`
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSessionNextShellEndedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextShellEnded struct {
	Data     OpenCodeSessionNextShellEndedData     `json:"data"`
	Durable  *OpenCodeSessionNextShellEndedDurable `json:"durable,omitempty"`
	ID       string                                `json:"id"`
	Location OpenCodeLocationRef                   `json:"location,omitempty"`
	Metadata map[string]any                        `json:"metadata,omitempty"`
	Type     string                                `json:"type"`
}

type OpenCodeSessionNextShellStartedData struct {
	CallID    string  `json:"callID"`
	Command   string  `json:"command"`
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSessionNextShellStartedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextShellStarted struct {
	Data     OpenCodeSessionNextShellStartedData     `json:"data"`
	Durable  *OpenCodeSessionNextShellStartedDurable `json:"durable,omitempty"`
	ID       string                                  `json:"id"`
	Location OpenCodeLocationRef                     `json:"location,omitempty"`
	Metadata map[string]any                          `json:"metadata,omitempty"`
	Type     string                                  `json:"type"`
}

type OpenCodeSessionNextStepEndedDataTokensCache struct {
	Read  float64 `json:"read"`
	Write float64 `json:"write"`
}

type OpenCodeSessionNextStepEndedDataTokens struct {
	Cache     OpenCodeSessionNextStepEndedDataTokensCache `json:"cache"`
	Input     float64                                     `json:"input"`
	Output    float64                                     `json:"output"`
	Reasoning float64                                     `json:"reasoning"`
}

type OpenCodeSessionNextStepEndedData struct {
	AssistantMessageID string                                 `json:"assistantMessageID"`
	Cost               float64                                `json:"cost"`
	Files              []string                               `json:"files,omitempty"`
	Finish             string                                 `json:"finish"`
	SessionID          string                                 `json:"sessionID"`
	Snapshot           string                                 `json:"snapshot,omitempty"`
	Timestamp          float64                                `json:"timestamp"`
	Tokens             OpenCodeSessionNextStepEndedDataTokens `json:"tokens"`
}

type OpenCodeSessionNextStepEndedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextStepEnded struct {
	Data     OpenCodeSessionNextStepEndedData     `json:"data"`
	Durable  *OpenCodeSessionNextStepEndedDurable `json:"durable,omitempty"`
	ID       string                               `json:"id"`
	Location OpenCodeLocationRef                  `json:"location,omitempty"`
	Metadata map[string]any                       `json:"metadata,omitempty"`
	Type     string                               `json:"type"`
}

type OpenCodeSessionNextStepFailedData struct {
	AssistantMessageID string                      `json:"assistantMessageID"`
	Error              OpenCodeSessionErrorUnknown `json:"error"`
	SessionID          string                      `json:"sessionID"`
	Timestamp          float64                     `json:"timestamp"`
}

type OpenCodeSessionNextStepFailedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextStepFailed struct {
	Data     OpenCodeSessionNextStepFailedData     `json:"data"`
	Durable  *OpenCodeSessionNextStepFailedDurable `json:"durable,omitempty"`
	ID       string                                `json:"id"`
	Location OpenCodeLocationRef                   `json:"location,omitempty"`
	Metadata map[string]any                        `json:"metadata,omitempty"`
	Type     string                                `json:"type"`
}

type OpenCodeSessionNextStepStartedData struct {
	Agent              string           `json:"agent"`
	AssistantMessageID string           `json:"assistantMessageID"`
	Model              OpenCodeModelRef `json:"model"`
	SessionID          string           `json:"sessionID"`
	Snapshot           string           `json:"snapshot,omitempty"`
	Timestamp          float64          `json:"timestamp"`
}

type OpenCodeSessionNextStepStartedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextStepStarted struct {
	Data     OpenCodeSessionNextStepStartedData     `json:"data"`
	Durable  *OpenCodeSessionNextStepStartedDurable `json:"durable,omitempty"`
	ID       string                                 `json:"id"`
	Location OpenCodeLocationRef                    `json:"location,omitempty"`
	Metadata map[string]any                         `json:"metadata,omitempty"`
	Type     string                                 `json:"type"`
}

type OpenCodeSessionNextSyntheticData struct {
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSessionNextSyntheticDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextSynthetic struct {
	Data     OpenCodeSessionNextSyntheticData     `json:"data"`
	Durable  *OpenCodeSessionNextSyntheticDurable `json:"durable,omitempty"`
	ID       string                               `json:"id"`
	Location OpenCodeLocationRef                  `json:"location,omitempty"`
	Metadata map[string]any                       `json:"metadata,omitempty"`
	Type     string                               `json:"type"`
}

type OpenCodeSessionNextTextDeltaData struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	Delta              string  `json:"delta"`
	SessionID          string  `json:"sessionID"`
	TextID             string  `json:"textID"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeSessionNextTextDeltaDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextTextDelta struct {
	Data     OpenCodeSessionNextTextDeltaData     `json:"data"`
	Durable  *OpenCodeSessionNextTextDeltaDurable `json:"durable,omitempty"`
	ID       string                               `json:"id"`
	Location OpenCodeLocationRef                  `json:"location,omitempty"`
	Metadata map[string]any                       `json:"metadata,omitempty"`
	Type     string                               `json:"type"`
}

type OpenCodeSessionNextTextEndedData struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	SessionID          string  `json:"sessionID"`
	Text               string  `json:"text"`
	TextID             string  `json:"textID"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeSessionNextTextEndedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextTextEnded struct {
	Data     OpenCodeSessionNextTextEndedData     `json:"data"`
	Durable  *OpenCodeSessionNextTextEndedDurable `json:"durable,omitempty"`
	ID       string                               `json:"id"`
	Location OpenCodeLocationRef                  `json:"location,omitempty"`
	Metadata map[string]any                       `json:"metadata,omitempty"`
	Type     string                               `json:"type"`
}

type OpenCodeSessionNextTextStartedData struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	SessionID          string  `json:"sessionID"`
	TextID             string  `json:"textID"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeSessionNextTextStartedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextTextStarted struct {
	Data     OpenCodeSessionNextTextStartedData     `json:"data"`
	Durable  *OpenCodeSessionNextTextStartedDurable `json:"durable,omitempty"`
	ID       string                                 `json:"id"`
	Location OpenCodeLocationRef                    `json:"location,omitempty"`
	Metadata map[string]any                         `json:"metadata,omitempty"`
	Type     string                                 `json:"type"`
}

type OpenCodeSessionNextToolCalledDataProvider struct {
	Executed bool                        `json:"executed"`
	Metadata OpenCodeLLMProviderMetadata `json:"metadata,omitempty"`
}

type OpenCodeSessionNextToolCalledData struct {
	AssistantMessageID string                                    `json:"assistantMessageID"`
	CallID             string                                    `json:"callID"`
	Input              map[string]any                            `json:"input"`
	Provider           OpenCodeSessionNextToolCalledDataProvider `json:"provider"`
	SessionID          string                                    `json:"sessionID"`
	Timestamp          float64                                   `json:"timestamp"`
	Tool               string                                    `json:"tool"`
}

type OpenCodeSessionNextToolCalledDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextToolCalled struct {
	Data     OpenCodeSessionNextToolCalledData     `json:"data"`
	Durable  *OpenCodeSessionNextToolCalledDurable `json:"durable,omitempty"`
	ID       string                                `json:"id"`
	Location OpenCodeLocationRef                   `json:"location,omitempty"`
	Metadata map[string]any                        `json:"metadata,omitempty"`
	Type     string                                `json:"type"`
}

type OpenCodeSessionNextToolFailedDataProvider struct {
	Executed bool                        `json:"executed"`
	Metadata OpenCodeLLMProviderMetadata `json:"metadata,omitempty"`
}

type OpenCodeSessionNextToolFailedData struct {
	AssistantMessageID string                                    `json:"assistantMessageID"`
	CallID             string                                    `json:"callID"`
	Error              OpenCodeSessionErrorUnknown               `json:"error"`
	Provider           OpenCodeSessionNextToolFailedDataProvider `json:"provider"`
	Result             any                                       `json:"result,omitempty"`
	SessionID          string                                    `json:"sessionID"`
	Timestamp          float64                                   `json:"timestamp"`
}

type OpenCodeSessionNextToolFailedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextToolFailed struct {
	Data     OpenCodeSessionNextToolFailedData     `json:"data"`
	Durable  *OpenCodeSessionNextToolFailedDurable `json:"durable,omitempty"`
	ID       string                                `json:"id"`
	Location OpenCodeLocationRef                   `json:"location,omitempty"`
	Metadata map[string]any                        `json:"metadata,omitempty"`
	Type     string                                `json:"type"`
}

type OpenCodeSessionNextToolInputDeltaData struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	CallID             string  `json:"callID"`
	Delta              string  `json:"delta"`
	SessionID          string  `json:"sessionID"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeSessionNextToolInputDeltaDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextToolInputDelta struct {
	Data     OpenCodeSessionNextToolInputDeltaData     `json:"data"`
	Durable  *OpenCodeSessionNextToolInputDeltaDurable `json:"durable,omitempty"`
	ID       string                                    `json:"id"`
	Location OpenCodeLocationRef                       `json:"location,omitempty"`
	Metadata map[string]any                            `json:"metadata,omitempty"`
	Type     string                                    `json:"type"`
}

type OpenCodeSessionNextToolInputEndedData struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	CallID             string  `json:"callID"`
	SessionID          string  `json:"sessionID"`
	Text               string  `json:"text"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeSessionNextToolInputEndedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextToolInputEnded struct {
	Data     OpenCodeSessionNextToolInputEndedData     `json:"data"`
	Durable  *OpenCodeSessionNextToolInputEndedDurable `json:"durable,omitempty"`
	ID       string                                    `json:"id"`
	Location OpenCodeLocationRef                       `json:"location,omitempty"`
	Metadata map[string]any                            `json:"metadata,omitempty"`
	Type     string                                    `json:"type"`
}

type OpenCodeSessionNextToolInputStartedData struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	CallID             string  `json:"callID"`
	Name               string  `json:"name"`
	SessionID          string  `json:"sessionID"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeSessionNextToolInputStartedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextToolInputStarted struct {
	Data     OpenCodeSessionNextToolInputStartedData     `json:"data"`
	Durable  *OpenCodeSessionNextToolInputStartedDurable `json:"durable,omitempty"`
	ID       string                                      `json:"id"`
	Location OpenCodeLocationRef                         `json:"location,omitempty"`
	Metadata map[string]any                              `json:"metadata,omitempty"`
	Type     string                                      `json:"type"`
}

type OpenCodeSessionNextToolProgressData struct {
	AssistantMessageID string                   `json:"assistantMessageID"`
	CallID             string                   `json:"callID"`
	Content            []OpenCodeLLMToolContent `json:"content"`
	SessionID          string                   `json:"sessionID"`
	Structured         map[string]any           `json:"structured"`
	Timestamp          float64                  `json:"timestamp"`
}

type OpenCodeSessionNextToolProgressDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextToolProgress struct {
	Data     OpenCodeSessionNextToolProgressData     `json:"data"`
	Durable  *OpenCodeSessionNextToolProgressDurable `json:"durable,omitempty"`
	ID       string                                  `json:"id"`
	Location OpenCodeLocationRef                     `json:"location,omitempty"`
	Metadata map[string]any                          `json:"metadata,omitempty"`
	Type     string                                  `json:"type"`
}

type OpenCodeSessionNextToolSuccessDataProvider struct {
	Executed bool                        `json:"executed"`
	Metadata OpenCodeLLMProviderMetadata `json:"metadata,omitempty"`
}

type OpenCodeSessionNextToolSuccessData struct {
	AssistantMessageID string                                     `json:"assistantMessageID"`
	CallID             string                                     `json:"callID"`
	Content            []OpenCodeLLMToolContent                   `json:"content"`
	OutputPaths        []string                                   `json:"outputPaths,omitempty"`
	Provider           OpenCodeSessionNextToolSuccessDataProvider `json:"provider"`
	Result             any                                        `json:"result,omitempty"`
	SessionID          string                                     `json:"sessionID"`
	Structured         map[string]any                             `json:"structured"`
	Timestamp          float64                                    `json:"timestamp"`
}

type OpenCodeSessionNextToolSuccessDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionNextToolSuccess struct {
	Data     OpenCodeSessionNextToolSuccessData     `json:"data"`
	Durable  *OpenCodeSessionNextToolSuccessDurable `json:"durable,omitempty"`
	ID       string                                 `json:"id"`
	Location OpenCodeLocationRef                    `json:"location,omitempty"`
	Metadata map[string]any                         `json:"metadata,omitempty"`
	Type     string                                 `json:"type"`
}

type OpenCodeSessionNotFoundError struct {
	Tag       string `json:"_tag"`
	Message   string `json:"message"`
	SessionID string `json:"sessionID"`
}

type OpenCodeSessionStatus any

type OpenCodeSessionUpdatedData struct {
	Info      OpenCodeSession `json:"info"`
	SessionID string          `json:"sessionID"`
}

type OpenCodeSessionUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionUpdated struct {
	Data     OpenCodeSessionUpdatedData     `json:"data"`
	Durable  *OpenCodeSessionUpdatedDurable `json:"durable,omitempty"`
	ID       string                         `json:"id"`
	Location OpenCodeLocationRef            `json:"location,omitempty"`
	Metadata map[string]any                 `json:"metadata,omitempty"`
	Type     string                         `json:"type"`
}

type OpenCodeSessionV2InfoTime struct {
	Archived float64 `json:"archived,omitempty"`
	Created  float64 `json:"created"`
	Updated  float64 `json:"updated"`
}

type OpenCodeSessionV2InfoTokensCache struct {
	Read  float64 `json:"read"`
	Write float64 `json:"write"`
}

type OpenCodeSessionV2InfoTokens struct {
	Cache     OpenCodeSessionV2InfoTokensCache `json:"cache"`
	Input     float64                          `json:"input"`
	Output    float64                          `json:"output"`
	Reasoning float64                          `json:"reasoning"`
}

type OpenCodeSessionV2Info struct {
	Agent     string                      `json:"agent,omitempty"`
	Cost      float64                     `json:"cost"`
	ID        string                      `json:"id"`
	Location  OpenCodeLocationRef         `json:"location"`
	Model     OpenCodeModelRef            `json:"model,omitempty"`
	ParentID  string                      `json:"parentID,omitempty"`
	ProjectID string                      `json:"projectID"`
	Revert    OpenCodeRevertState         `json:"revert,omitempty"`
	Subpath   string                      `json:"subpath,omitempty"`
	Time      OpenCodeSessionV2InfoTime   `json:"time"`
	Title     string                      `json:"title"`
	Tokens    OpenCodeSessionV2InfoTokens `json:"tokens"`
}

type OpenCodeSessionsResponseCursor struct {
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
}

type OpenCodeSessionsResponse struct {
	Cursor OpenCodeSessionsResponseCursor `json:"cursor"`
	Data   []OpenCodeSessionV2Info        `json:"data"`
}

type OpenCodeSkillV2DirectorySource struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type OpenCodeSkillV2EmbeddedSource struct {
	Skill OpenCodeSkillV2Info `json:"skill"`
	Type  string              `json:"type"`
}

type OpenCodeSkillV2Info struct {
	Content     string `json:"content"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location"`
	Name        string `json:"name"`
	Slash       bool   `json:"slash,omitempty"`
}

type OpenCodeSkillV2Source any

type OpenCodeSkillV2UrlSource struct {
	Type string `json:"type"`
	Url  string `json:"url"`
}

type OpenCodeSnapshotFileDiff struct {
	Additions float64 `json:"additions"`
	Deletions float64 `json:"deletions"`
	File      string  `json:"file,omitempty"`
	Patch     string  `json:"patch,omitempty"`
	Status    string  `json:"status,omitempty"`
}

type OpenCodeSnapshotPart struct {
	ID        string `json:"id"`
	MessageID string `json:"messageID"`
	SessionID string `json:"sessionID"`
	Snapshot  string `json:"snapshot"`
	Type      string `json:"type"`
}

type OpenCodeStepFinishPartTokensCache struct {
	Read  float64 `json:"read"`
	Write float64 `json:"write"`
}

type OpenCodeStepFinishPartTokens struct {
	Cache     OpenCodeStepFinishPartTokensCache `json:"cache"`
	Input     float64                           `json:"input"`
	Output    float64                           `json:"output"`
	Reasoning float64                           `json:"reasoning"`
	Total     float64                           `json:"total,omitempty"`
}

type OpenCodeStepFinishPart struct {
	Cost      float64                      `json:"cost"`
	ID        string                       `json:"id"`
	MessageID string                       `json:"messageID"`
	Reason    string                       `json:"reason"`
	SessionID string                       `json:"sessionID"`
	Snapshot  string                       `json:"snapshot,omitempty"`
	Tokens    OpenCodeStepFinishPartTokens `json:"tokens"`
	Type      string                       `json:"type"`
}

type OpenCodeStepStartPart struct {
	ID        string `json:"id"`
	MessageID string `json:"messageID"`
	SessionID string `json:"sessionID"`
	Snapshot  string `json:"snapshot,omitempty"`
	Type      string `json:"type"`
}

type OpenCodeStructuredOutputErrorData struct {
	Message string `json:"message"`
	Retries int64  `json:"retries"`
}

type OpenCodeStructuredOutputError struct {
	Data OpenCodeStructuredOutputErrorData `json:"data"`
	Name string                            `json:"name"`
}

type OpenCodeSubtaskPartModel struct {
	ModelID    string `json:"modelID"`
	ProviderID string `json:"providerID"`
}

type OpenCodeSubtaskPart struct {
	Agent       string                    `json:"agent"`
	Command     string                    `json:"command,omitempty"`
	Description string                    `json:"description"`
	ID          string                    `json:"id"`
	MessageID   string                    `json:"messageID"`
	Model       *OpenCodeSubtaskPartModel `json:"model,omitempty"`
	Prompt      string                    `json:"prompt"`
	SessionID   string                    `json:"sessionID"`
	Type        string                    `json:"type"`
}

type OpenCodeSubtaskPartInputModel struct {
	ModelID    string `json:"modelID"`
	ProviderID string `json:"providerID"`
}

type OpenCodeSubtaskPartInput struct {
	Agent       string                         `json:"agent"`
	Command     string                         `json:"command,omitempty"`
	Description string                         `json:"description"`
	ID          string                         `json:"id,omitempty"`
	Model       *OpenCodeSubtaskPartInputModel `json:"model,omitempty"`
	Prompt      string                         `json:"prompt"`
	Type        string                         `json:"type"`
}

type OpenCodeSymbolLocation struct {
	Range OpenCodeRange `json:"range"`
	Uri   string        `json:"uri"`
}

type OpenCodeSymbol struct {
	Kind     int64                  `json:"kind"`
	Location OpenCodeSymbolLocation `json:"location"`
	Name     string                 `json:"name"`
}

type OpenCodeSymbolSource struct {
	Kind  int64                      `json:"kind"`
	Name  string                     `json:"name"`
	Path  string                     `json:"path"`
	Range OpenCodeRange              `json:"range"`
	Text  OpenCodeFilePartSourceText `json:"text"`
	Type  string                     `json:"type"`
}

type OpenCodeSyncEventMessagePartRemovedSyncEventData struct {
	MessageID string `json:"messageID"`
	PartID    string `json:"partID"`
	SessionID string `json:"sessionID"`
}

type OpenCodeSyncEventMessagePartRemovedSyncEvent struct {
	AggregateID string                                           `json:"aggregateID"`
	Data        OpenCodeSyncEventMessagePartRemovedSyncEventData `json:"data"`
	ID          string                                           `json:"id"`
	Seq         float64                                          `json:"seq"`
	Type        string                                           `json:"type"`
}

type OpenCodeSyncEventMessagePartRemoved struct {
	ID        string                                       `json:"id"`
	SyncEvent OpenCodeSyncEventMessagePartRemovedSyncEvent `json:"syncEvent"`
	Type      string                                       `json:"type"`
}

type OpenCodeSyncEventMessagePartUpdatedSyncEventData struct {
	Part      OpenCodePart `json:"part"`
	SessionID string       `json:"sessionID"`
	Time      float64      `json:"time"`
}

type OpenCodeSyncEventMessagePartUpdatedSyncEvent struct {
	AggregateID string                                           `json:"aggregateID"`
	Data        OpenCodeSyncEventMessagePartUpdatedSyncEventData `json:"data"`
	ID          string                                           `json:"id"`
	Seq         float64                                          `json:"seq"`
	Type        string                                           `json:"type"`
}

type OpenCodeSyncEventMessagePartUpdated struct {
	ID        string                                       `json:"id"`
	SyncEvent OpenCodeSyncEventMessagePartUpdatedSyncEvent `json:"syncEvent"`
	Type      string                                       `json:"type"`
}

type OpenCodeSyncEventMessageRemovedSyncEventData struct {
	MessageID string `json:"messageID"`
	SessionID string `json:"sessionID"`
}

type OpenCodeSyncEventMessageRemovedSyncEvent struct {
	AggregateID string                                       `json:"aggregateID"`
	Data        OpenCodeSyncEventMessageRemovedSyncEventData `json:"data"`
	ID          string                                       `json:"id"`
	Seq         float64                                      `json:"seq"`
	Type        string                                       `json:"type"`
}

type OpenCodeSyncEventMessageRemoved struct {
	ID        string                                   `json:"id"`
	SyncEvent OpenCodeSyncEventMessageRemovedSyncEvent `json:"syncEvent"`
	Type      string                                   `json:"type"`
}

type OpenCodeSyncEventMessageUpdatedSyncEventData struct {
	Info      OpenCodeMessage `json:"info"`
	SessionID string          `json:"sessionID"`
}

type OpenCodeSyncEventMessageUpdatedSyncEvent struct {
	AggregateID string                                       `json:"aggregateID"`
	Data        OpenCodeSyncEventMessageUpdatedSyncEventData `json:"data"`
	ID          string                                       `json:"id"`
	Seq         float64                                      `json:"seq"`
	Type        string                                       `json:"type"`
}

type OpenCodeSyncEventMessageUpdated struct {
	ID        string                                   `json:"id"`
	SyncEvent OpenCodeSyncEventMessageUpdatedSyncEvent `json:"syncEvent"`
	Type      string                                   `json:"type"`
}

type OpenCodeSyncEventSessionCreatedSyncEventData struct {
	Info      OpenCodeSession `json:"info"`
	SessionID string          `json:"sessionID"`
}

type OpenCodeSyncEventSessionCreatedSyncEvent struct {
	AggregateID string                                       `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionCreatedSyncEventData `json:"data"`
	ID          string                                       `json:"id"`
	Seq         float64                                      `json:"seq"`
	Type        string                                       `json:"type"`
}

type OpenCodeSyncEventSessionCreated struct {
	ID        string                                   `json:"id"`
	SyncEvent OpenCodeSyncEventSessionCreatedSyncEvent `json:"syncEvent"`
	Type      string                                   `json:"type"`
}

type OpenCodeSyncEventSessionDeletedSyncEventData struct {
	Info      OpenCodeSession `json:"info"`
	SessionID string          `json:"sessionID"`
}

type OpenCodeSyncEventSessionDeletedSyncEvent struct {
	AggregateID string                                       `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionDeletedSyncEventData `json:"data"`
	ID          string                                       `json:"id"`
	Seq         float64                                      `json:"seq"`
	Type        string                                       `json:"type"`
}

type OpenCodeSyncEventSessionDeleted struct {
	ID        string                                   `json:"id"`
	SyncEvent OpenCodeSyncEventSessionDeletedSyncEvent `json:"syncEvent"`
	Type      string                                   `json:"type"`
}

type OpenCodeSyncEventSessionNextAgentSwitchedSyncEventData struct {
	Agent     string  `json:"agent"`
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextAgentSwitchedSyncEvent struct {
	AggregateID string                                                 `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextAgentSwitchedSyncEventData `json:"data"`
	ID          string                                                 `json:"id"`
	Seq         float64                                                `json:"seq"`
	Type        string                                                 `json:"type"`
}

type OpenCodeSyncEventSessionNextAgentSwitched struct {
	ID        string                                             `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextAgentSwitchedSyncEvent `json:"syncEvent"`
	Type      string                                             `json:"type"`
}

type OpenCodeSyncEventSessionNextCompactionEndedSyncEventData struct {
	MessageID string  `json:"messageID"`
	Reason    string  `json:"reason"`
	Recent    string  `json:"recent"`
	SessionID string  `json:"sessionID"`
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextCompactionEndedSyncEvent struct {
	AggregateID string                                                   `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextCompactionEndedSyncEventData `json:"data"`
	ID          string                                                   `json:"id"`
	Seq         float64                                                  `json:"seq"`
	Type        string                                                   `json:"type"`
}

type OpenCodeSyncEventSessionNextCompactionEnded struct {
	ID        string                                               `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextCompactionEndedSyncEvent `json:"syncEvent"`
	Type      string                                               `json:"type"`
}

type OpenCodeSyncEventSessionNextCompactionStartedSyncEventData struct {
	MessageID string  `json:"messageID"`
	Reason    string  `json:"reason"`
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextCompactionStartedSyncEvent struct {
	AggregateID string                                                     `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextCompactionStartedSyncEventData `json:"data"`
	ID          string                                                     `json:"id"`
	Seq         float64                                                    `json:"seq"`
	Type        string                                                     `json:"type"`
}

type OpenCodeSyncEventSessionNextCompactionStarted struct {
	ID        string                                                 `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextCompactionStartedSyncEvent `json:"syncEvent"`
	Type      string                                                 `json:"type"`
}

type OpenCodeSyncEventSessionNextContextUpdatedSyncEventData struct {
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextContextUpdatedSyncEvent struct {
	AggregateID string                                                  `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextContextUpdatedSyncEventData `json:"data"`
	ID          string                                                  `json:"id"`
	Seq         float64                                                 `json:"seq"`
	Type        string                                                  `json:"type"`
}

type OpenCodeSyncEventSessionNextContextUpdated struct {
	ID        string                                              `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextContextUpdatedSyncEvent `json:"syncEvent"`
	Type      string                                              `json:"type"`
}

type OpenCodeSyncEventSessionNextModelSwitchedSyncEventData struct {
	MessageID string           `json:"messageID"`
	Model     OpenCodeModelRef `json:"model"`
	SessionID string           `json:"sessionID"`
	Timestamp float64          `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextModelSwitchedSyncEvent struct {
	AggregateID string                                                 `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextModelSwitchedSyncEventData `json:"data"`
	ID          string                                                 `json:"id"`
	Seq         float64                                                `json:"seq"`
	Type        string                                                 `json:"type"`
}

type OpenCodeSyncEventSessionNextModelSwitched struct {
	ID        string                                             `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextModelSwitchedSyncEvent `json:"syncEvent"`
	Type      string                                             `json:"type"`
}

type OpenCodeSyncEventSessionNextMovedSyncEventData struct {
	Location     OpenCodeLocationRef `json:"location"`
	SessionID    string              `json:"sessionID"`
	Subdirectory string              `json:"subdirectory,omitempty"`
	Timestamp    float64             `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextMovedSyncEvent struct {
	AggregateID string                                         `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextMovedSyncEventData `json:"data"`
	ID          string                                         `json:"id"`
	Seq         float64                                        `json:"seq"`
	Type        string                                         `json:"type"`
}

type OpenCodeSyncEventSessionNextMoved struct {
	ID        string                                     `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextMovedSyncEvent `json:"syncEvent"`
	Type      string                                     `json:"type"`
}

type OpenCodeSyncEventSessionNextPromptAdmittedSyncEventData struct {
	Delivery  string         `json:"delivery"`
	MessageID string         `json:"messageID"`
	Prompt    OpenCodePrompt `json:"prompt"`
	SessionID string         `json:"sessionID"`
	Timestamp float64        `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextPromptAdmittedSyncEvent struct {
	AggregateID string                                                  `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextPromptAdmittedSyncEventData `json:"data"`
	ID          string                                                  `json:"id"`
	Seq         float64                                                 `json:"seq"`
	Type        string                                                  `json:"type"`
}

type OpenCodeSyncEventSessionNextPromptAdmitted struct {
	ID        string                                              `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextPromptAdmittedSyncEvent `json:"syncEvent"`
	Type      string                                              `json:"type"`
}

type OpenCodeSyncEventSessionNextPromptedSyncEventData struct {
	Delivery  string         `json:"delivery"`
	MessageID string         `json:"messageID"`
	Prompt    OpenCodePrompt `json:"prompt"`
	SessionID string         `json:"sessionID"`
	Timestamp float64        `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextPromptedSyncEvent struct {
	AggregateID string                                            `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextPromptedSyncEventData `json:"data"`
	ID          string                                            `json:"id"`
	Seq         float64                                           `json:"seq"`
	Type        string                                            `json:"type"`
}

type OpenCodeSyncEventSessionNextPrompted struct {
	ID        string                                        `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextPromptedSyncEvent `json:"syncEvent"`
	Type      string                                        `json:"type"`
}

type OpenCodeSyncEventSessionNextReasoningEndedSyncEventData struct {
	AssistantMessageID string                      `json:"assistantMessageID"`
	ProviderMetadata   OpenCodeLLMProviderMetadata `json:"providerMetadata,omitempty"`
	ReasoningID        string                      `json:"reasoningID"`
	SessionID          string                      `json:"sessionID"`
	Text               string                      `json:"text"`
	Timestamp          float64                     `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextReasoningEndedSyncEvent struct {
	AggregateID string                                                  `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextReasoningEndedSyncEventData `json:"data"`
	ID          string                                                  `json:"id"`
	Seq         float64                                                 `json:"seq"`
	Type        string                                                  `json:"type"`
}

type OpenCodeSyncEventSessionNextReasoningEnded struct {
	ID        string                                              `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextReasoningEndedSyncEvent `json:"syncEvent"`
	Type      string                                              `json:"type"`
}

type OpenCodeSyncEventSessionNextReasoningStartedSyncEventData struct {
	AssistantMessageID string                      `json:"assistantMessageID"`
	ProviderMetadata   OpenCodeLLMProviderMetadata `json:"providerMetadata,omitempty"`
	ReasoningID        string                      `json:"reasoningID"`
	SessionID          string                      `json:"sessionID"`
	Timestamp          float64                     `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextReasoningStartedSyncEvent struct {
	AggregateID string                                                    `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextReasoningStartedSyncEventData `json:"data"`
	ID          string                                                    `json:"id"`
	Seq         float64                                                   `json:"seq"`
	Type        string                                                    `json:"type"`
}

type OpenCodeSyncEventSessionNextReasoningStarted struct {
	ID        string                                                `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextReasoningStartedSyncEvent `json:"syncEvent"`
	Type      string                                                `json:"type"`
}

type OpenCodeSyncEventSessionNextRetriedSyncEventData struct {
	Attempt   float64                       `json:"attempt"`
	Error     OpenCodeSessionNextRetryError `json:"error"`
	SessionID string                        `json:"sessionID"`
	Timestamp float64                       `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextRetriedSyncEvent struct {
	AggregateID string                                           `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextRetriedSyncEventData `json:"data"`
	ID          string                                           `json:"id"`
	Seq         float64                                          `json:"seq"`
	Type        string                                           `json:"type"`
}

type OpenCodeSyncEventSessionNextRetried struct {
	ID        string                                       `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextRetriedSyncEvent `json:"syncEvent"`
	Type      string                                       `json:"type"`
}

type OpenCodeSyncEventSessionNextRevertClearedSyncEventData struct {
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextRevertClearedSyncEvent struct {
	AggregateID string                                                 `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextRevertClearedSyncEventData `json:"data"`
	ID          string                                                 `json:"id"`
	Seq         float64                                                `json:"seq"`
	Type        string                                                 `json:"type"`
}

type OpenCodeSyncEventSessionNextRevertCleared struct {
	ID        string                                             `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextRevertClearedSyncEvent `json:"syncEvent"`
	Type      string                                             `json:"type"`
}

type OpenCodeSyncEventSessionNextRevertCommittedSyncEventData struct {
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextRevertCommittedSyncEvent struct {
	AggregateID string                                                   `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextRevertCommittedSyncEventData `json:"data"`
	ID          string                                                   `json:"id"`
	Seq         float64                                                  `json:"seq"`
	Type        string                                                   `json:"type"`
}

type OpenCodeSyncEventSessionNextRevertCommitted struct {
	ID        string                                               `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextRevertCommittedSyncEvent `json:"syncEvent"`
	Type      string                                               `json:"type"`
}

type OpenCodeSyncEventSessionNextRevertStagedSyncEventData struct {
	Revert    OpenCodeRevertState `json:"revert"`
	SessionID string              `json:"sessionID"`
	Timestamp float64             `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextRevertStagedSyncEvent struct {
	AggregateID string                                                `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextRevertStagedSyncEventData `json:"data"`
	ID          string                                                `json:"id"`
	Seq         float64                                               `json:"seq"`
	Type        string                                                `json:"type"`
}

type OpenCodeSyncEventSessionNextRevertStaged struct {
	ID        string                                            `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextRevertStagedSyncEvent `json:"syncEvent"`
	Type      string                                            `json:"type"`
}

type OpenCodeSyncEventSessionNextShellEndedSyncEventData struct {
	CallID    string  `json:"callID"`
	Output    string  `json:"output"`
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextShellEndedSyncEvent struct {
	AggregateID string                                              `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextShellEndedSyncEventData `json:"data"`
	ID          string                                              `json:"id"`
	Seq         float64                                             `json:"seq"`
	Type        string                                              `json:"type"`
}

type OpenCodeSyncEventSessionNextShellEnded struct {
	ID        string                                          `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextShellEndedSyncEvent `json:"syncEvent"`
	Type      string                                          `json:"type"`
}

type OpenCodeSyncEventSessionNextShellStartedSyncEventData struct {
	CallID    string  `json:"callID"`
	Command   string  `json:"command"`
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextShellStartedSyncEvent struct {
	AggregateID string                                                `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextShellStartedSyncEventData `json:"data"`
	ID          string                                                `json:"id"`
	Seq         float64                                               `json:"seq"`
	Type        string                                                `json:"type"`
}

type OpenCodeSyncEventSessionNextShellStarted struct {
	ID        string                                            `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextShellStartedSyncEvent `json:"syncEvent"`
	Type      string                                            `json:"type"`
}

type OpenCodeSyncEventSessionNextStepEndedSyncEventDataTokensCache struct {
	Read  float64 `json:"read"`
	Write float64 `json:"write"`
}

type OpenCodeSyncEventSessionNextStepEndedSyncEventDataTokens struct {
	Cache     OpenCodeSyncEventSessionNextStepEndedSyncEventDataTokensCache `json:"cache"`
	Input     float64                                                       `json:"input"`
	Output    float64                                                       `json:"output"`
	Reasoning float64                                                       `json:"reasoning"`
}

type OpenCodeSyncEventSessionNextStepEndedSyncEventData struct {
	AssistantMessageID string                                                   `json:"assistantMessageID"`
	Cost               float64                                                  `json:"cost"`
	Files              []string                                                 `json:"files,omitempty"`
	Finish             string                                                   `json:"finish"`
	SessionID          string                                                   `json:"sessionID"`
	Snapshot           string                                                   `json:"snapshot,omitempty"`
	Timestamp          float64                                                  `json:"timestamp"`
	Tokens             OpenCodeSyncEventSessionNextStepEndedSyncEventDataTokens `json:"tokens"`
}

type OpenCodeSyncEventSessionNextStepEndedSyncEvent struct {
	AggregateID string                                             `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextStepEndedSyncEventData `json:"data"`
	ID          string                                             `json:"id"`
	Seq         float64                                            `json:"seq"`
	Type        string                                             `json:"type"`
}

type OpenCodeSyncEventSessionNextStepEnded struct {
	ID        string                                         `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextStepEndedSyncEvent `json:"syncEvent"`
	Type      string                                         `json:"type"`
}

type OpenCodeSyncEventSessionNextStepFailedSyncEventData struct {
	AssistantMessageID string                      `json:"assistantMessageID"`
	Error              OpenCodeSessionErrorUnknown `json:"error"`
	SessionID          string                      `json:"sessionID"`
	Timestamp          float64                     `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextStepFailedSyncEvent struct {
	AggregateID string                                              `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextStepFailedSyncEventData `json:"data"`
	ID          string                                              `json:"id"`
	Seq         float64                                             `json:"seq"`
	Type        string                                              `json:"type"`
}

type OpenCodeSyncEventSessionNextStepFailed struct {
	ID        string                                          `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextStepFailedSyncEvent `json:"syncEvent"`
	Type      string                                          `json:"type"`
}

type OpenCodeSyncEventSessionNextStepStartedSyncEventData struct {
	Agent              string           `json:"agent"`
	AssistantMessageID string           `json:"assistantMessageID"`
	Model              OpenCodeModelRef `json:"model"`
	SessionID          string           `json:"sessionID"`
	Snapshot           string           `json:"snapshot,omitempty"`
	Timestamp          float64          `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextStepStartedSyncEvent struct {
	AggregateID string                                               `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextStepStartedSyncEventData `json:"data"`
	ID          string                                               `json:"id"`
	Seq         float64                                              `json:"seq"`
	Type        string                                               `json:"type"`
}

type OpenCodeSyncEventSessionNextStepStarted struct {
	ID        string                                           `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextStepStartedSyncEvent `json:"syncEvent"`
	Type      string                                           `json:"type"`
}

type OpenCodeSyncEventSessionNextSyntheticSyncEventData struct {
	MessageID string  `json:"messageID"`
	SessionID string  `json:"sessionID"`
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextSyntheticSyncEvent struct {
	AggregateID string                                             `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextSyntheticSyncEventData `json:"data"`
	ID          string                                             `json:"id"`
	Seq         float64                                            `json:"seq"`
	Type        string                                             `json:"type"`
}

type OpenCodeSyncEventSessionNextSynthetic struct {
	ID        string                                         `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextSyntheticSyncEvent `json:"syncEvent"`
	Type      string                                         `json:"type"`
}

type OpenCodeSyncEventSessionNextTextEndedSyncEventData struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	SessionID          string  `json:"sessionID"`
	Text               string  `json:"text"`
	TextID             string  `json:"textID"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextTextEndedSyncEvent struct {
	AggregateID string                                             `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextTextEndedSyncEventData `json:"data"`
	ID          string                                             `json:"id"`
	Seq         float64                                            `json:"seq"`
	Type        string                                             `json:"type"`
}

type OpenCodeSyncEventSessionNextTextEnded struct {
	ID        string                                         `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextTextEndedSyncEvent `json:"syncEvent"`
	Type      string                                         `json:"type"`
}

type OpenCodeSyncEventSessionNextTextStartedSyncEventData struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	SessionID          string  `json:"sessionID"`
	TextID             string  `json:"textID"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextTextStartedSyncEvent struct {
	AggregateID string                                               `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextTextStartedSyncEventData `json:"data"`
	ID          string                                               `json:"id"`
	Seq         float64                                              `json:"seq"`
	Type        string                                               `json:"type"`
}

type OpenCodeSyncEventSessionNextTextStarted struct {
	ID        string                                           `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextTextStartedSyncEvent `json:"syncEvent"`
	Type      string                                           `json:"type"`
}

type OpenCodeSyncEventSessionNextToolCalledSyncEventDataProvider struct {
	Executed bool                        `json:"executed"`
	Metadata OpenCodeLLMProviderMetadata `json:"metadata,omitempty"`
}

type OpenCodeSyncEventSessionNextToolCalledSyncEventData struct {
	AssistantMessageID string                                                      `json:"assistantMessageID"`
	CallID             string                                                      `json:"callID"`
	Input              map[string]any                                              `json:"input"`
	Provider           OpenCodeSyncEventSessionNextToolCalledSyncEventDataProvider `json:"provider"`
	SessionID          string                                                      `json:"sessionID"`
	Timestamp          float64                                                     `json:"timestamp"`
	Tool               string                                                      `json:"tool"`
}

type OpenCodeSyncEventSessionNextToolCalledSyncEvent struct {
	AggregateID string                                              `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextToolCalledSyncEventData `json:"data"`
	ID          string                                              `json:"id"`
	Seq         float64                                             `json:"seq"`
	Type        string                                              `json:"type"`
}

type OpenCodeSyncEventSessionNextToolCalled struct {
	ID        string                                          `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextToolCalledSyncEvent `json:"syncEvent"`
	Type      string                                          `json:"type"`
}

type OpenCodeSyncEventSessionNextToolFailedSyncEventDataProvider struct {
	Executed bool                        `json:"executed"`
	Metadata OpenCodeLLMProviderMetadata `json:"metadata,omitempty"`
}

type OpenCodeSyncEventSessionNextToolFailedSyncEventData struct {
	AssistantMessageID string                                                      `json:"assistantMessageID"`
	CallID             string                                                      `json:"callID"`
	Error              OpenCodeSessionErrorUnknown                                 `json:"error"`
	Provider           OpenCodeSyncEventSessionNextToolFailedSyncEventDataProvider `json:"provider"`
	Result             any                                                         `json:"result,omitempty"`
	SessionID          string                                                      `json:"sessionID"`
	Timestamp          float64                                                     `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextToolFailedSyncEvent struct {
	AggregateID string                                              `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextToolFailedSyncEventData `json:"data"`
	ID          string                                              `json:"id"`
	Seq         float64                                             `json:"seq"`
	Type        string                                              `json:"type"`
}

type OpenCodeSyncEventSessionNextToolFailed struct {
	ID        string                                          `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextToolFailedSyncEvent `json:"syncEvent"`
	Type      string                                          `json:"type"`
}

type OpenCodeSyncEventSessionNextToolInputEndedSyncEventData struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	CallID             string  `json:"callID"`
	SessionID          string  `json:"sessionID"`
	Text               string  `json:"text"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextToolInputEndedSyncEvent struct {
	AggregateID string                                                  `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextToolInputEndedSyncEventData `json:"data"`
	ID          string                                                  `json:"id"`
	Seq         float64                                                 `json:"seq"`
	Type        string                                                  `json:"type"`
}

type OpenCodeSyncEventSessionNextToolInputEnded struct {
	ID        string                                              `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextToolInputEndedSyncEvent `json:"syncEvent"`
	Type      string                                              `json:"type"`
}

type OpenCodeSyncEventSessionNextToolInputStartedSyncEventData struct {
	AssistantMessageID string  `json:"assistantMessageID"`
	CallID             string  `json:"callID"`
	Name               string  `json:"name"`
	SessionID          string  `json:"sessionID"`
	Timestamp          float64 `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextToolInputStartedSyncEvent struct {
	AggregateID string                                                    `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextToolInputStartedSyncEventData `json:"data"`
	ID          string                                                    `json:"id"`
	Seq         float64                                                   `json:"seq"`
	Type        string                                                    `json:"type"`
}

type OpenCodeSyncEventSessionNextToolInputStarted struct {
	ID        string                                                `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextToolInputStartedSyncEvent `json:"syncEvent"`
	Type      string                                                `json:"type"`
}

type OpenCodeSyncEventSessionNextToolProgressSyncEventData struct {
	AssistantMessageID string                   `json:"assistantMessageID"`
	CallID             string                   `json:"callID"`
	Content            []OpenCodeLLMToolContent `json:"content"`
	SessionID          string                   `json:"sessionID"`
	Structured         map[string]any           `json:"structured"`
	Timestamp          float64                  `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextToolProgressSyncEvent struct {
	AggregateID string                                                `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextToolProgressSyncEventData `json:"data"`
	ID          string                                                `json:"id"`
	Seq         float64                                               `json:"seq"`
	Type        string                                                `json:"type"`
}

type OpenCodeSyncEventSessionNextToolProgress struct {
	ID        string                                            `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextToolProgressSyncEvent `json:"syncEvent"`
	Type      string                                            `json:"type"`
}

type OpenCodeSyncEventSessionNextToolSuccessSyncEventDataProvider struct {
	Executed bool                        `json:"executed"`
	Metadata OpenCodeLLMProviderMetadata `json:"metadata,omitempty"`
}

type OpenCodeSyncEventSessionNextToolSuccessSyncEventData struct {
	AssistantMessageID string                                                       `json:"assistantMessageID"`
	CallID             string                                                       `json:"callID"`
	Content            []OpenCodeLLMToolContent                                     `json:"content"`
	OutputPaths        []string                                                     `json:"outputPaths,omitempty"`
	Provider           OpenCodeSyncEventSessionNextToolSuccessSyncEventDataProvider `json:"provider"`
	Result             any                                                          `json:"result,omitempty"`
	SessionID          string                                                       `json:"sessionID"`
	Structured         map[string]any                                               `json:"structured"`
	Timestamp          float64                                                      `json:"timestamp"`
}

type OpenCodeSyncEventSessionNextToolSuccessSyncEvent struct {
	AggregateID string                                               `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionNextToolSuccessSyncEventData `json:"data"`
	ID          string                                               `json:"id"`
	Seq         float64                                              `json:"seq"`
	Type        string                                               `json:"type"`
}

type OpenCodeSyncEventSessionNextToolSuccess struct {
	ID        string                                           `json:"id"`
	SyncEvent OpenCodeSyncEventSessionNextToolSuccessSyncEvent `json:"syncEvent"`
	Type      string                                           `json:"type"`
}

type OpenCodeSyncEventSessionUpdatedSyncEventData struct {
	Info      OpenCodeSession `json:"info"`
	SessionID string          `json:"sessionID"`
}

type OpenCodeSyncEventSessionUpdatedSyncEvent struct {
	AggregateID string                                       `json:"aggregateID"`
	Data        OpenCodeSyncEventSessionUpdatedSyncEventData `json:"data"`
	ID          string                                       `json:"id"`
	Seq         float64                                      `json:"seq"`
	Type        string                                       `json:"type"`
}

type OpenCodeSyncEventSessionUpdated struct {
	ID        string                                   `json:"id"`
	SyncEvent OpenCodeSyncEventSessionUpdatedSyncEvent `json:"syncEvent"`
	Type      string                                   `json:"type"`
}

type OpenCodeTextPartTime struct {
	End   int64 `json:"end,omitempty"`
	Start int64 `json:"start"`
}

type OpenCodeTextPart struct {
	ID        string                `json:"id"`
	Ignored   bool                  `json:"ignored,omitempty"`
	MessageID string                `json:"messageID"`
	Metadata  map[string]any        `json:"metadata,omitempty"`
	SessionID string                `json:"sessionID"`
	Synthetic bool                  `json:"synthetic,omitempty"`
	Text      string                `json:"text"`
	Time      *OpenCodeTextPartTime `json:"time,omitempty"`
	Type      string                `json:"type"`
}

type OpenCodeTextPartInputTime struct {
	End   int64 `json:"end,omitempty"`
	Start int64 `json:"start"`
}

type OpenCodeTextPartInput struct {
	ID        string                     `json:"id,omitempty"`
	Ignored   bool                       `json:"ignored,omitempty"`
	Metadata  map[string]any             `json:"metadata,omitempty"`
	Synthetic bool                       `json:"synthetic,omitempty"`
	Text      string                     `json:"text"`
	Time      *OpenCodeTextPartInputTime `json:"time,omitempty"`
	Type      string                     `json:"type"`
}

type OpenCodeTodo struct {
	Content  string `json:"content"`
	Priority string `json:"priority"`
	Status   string `json:"status"`
}

type OpenCodeTodoUpdatedData struct {
	SessionID string         `json:"sessionID"`
	Todos     []OpenCodeTodo `json:"todos"`
}

type OpenCodeTodoUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeTodoUpdated struct {
	Data     OpenCodeTodoUpdatedData     `json:"data"`
	Durable  *OpenCodeTodoUpdatedDurable `json:"durable,omitempty"`
	ID       string                      `json:"id"`
	Location OpenCodeLocationRef         `json:"location,omitempty"`
	Metadata map[string]any              `json:"metadata,omitempty"`
	Type     string                      `json:"type"`
}

type OpenCodeToolFileContent struct {
	Mime string `json:"mime"`
	Name string `json:"name,omitempty"`
	Type string `json:"type"`
	Uri  string `json:"uri"`
}

type OpenCodeToolIDs []string

type OpenCodeToolList []OpenCodeToolListItem

type OpenCodeToolListItem struct {
	Description string `json:"description"`
	ID          string `json:"id"`
	Parameters  any    `json:"parameters"`
}

type OpenCodeToolPart struct {
	CallID    string            `json:"callID"`
	ID        string            `json:"id"`
	MessageID string            `json:"messageID"`
	Metadata  map[string]any    `json:"metadata,omitempty"`
	SessionID string            `json:"sessionID"`
	State     OpenCodeToolState `json:"state"`
	Tool      string            `json:"tool"`
	Type      string            `json:"type"`
}

type OpenCodeToolState any

type OpenCodeToolStateCompletedTime struct {
	Compacted int64 `json:"compacted,omitempty"`
	End       int64 `json:"end"`
	Start     int64 `json:"start"`
}

type OpenCodeToolStateCompleted struct {
	Attachments []OpenCodeFilePart             `json:"attachments,omitempty"`
	Input       map[string]any                 `json:"input"`
	Metadata    map[string]any                 `json:"metadata"`
	Output      string                         `json:"output"`
	Status      string                         `json:"status"`
	Time        OpenCodeToolStateCompletedTime `json:"time"`
	Title       string                         `json:"title"`
}

type OpenCodeToolStateErrorTime struct {
	End   int64 `json:"end"`
	Start int64 `json:"start"`
}

type OpenCodeToolStateError struct {
	Error    string                     `json:"error"`
	Input    map[string]any             `json:"input"`
	Metadata map[string]any             `json:"metadata,omitempty"`
	Status   string                     `json:"status"`
	Time     OpenCodeToolStateErrorTime `json:"time"`
}

type OpenCodeToolStatePending struct {
	Input  map[string]any `json:"input"`
	Raw    string         `json:"raw"`
	Status string         `json:"status"`
}

type OpenCodeToolStateRunningTime struct {
	Start int64 `json:"start"`
}

type OpenCodeToolStateRunning struct {
	Input    map[string]any               `json:"input"`
	Metadata map[string]any               `json:"metadata,omitempty"`
	Status   string                       `json:"status"`
	Time     OpenCodeToolStateRunningTime `json:"time"`
	Title    string                       `json:"title,omitempty"`
}

type OpenCodeToolTextContent struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type OpenCodeTuiCommandExecuteData struct {
	Command any `json:"command"`
}

type OpenCodeTuiCommandExecuteDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeTuiCommandExecute struct {
	Data     OpenCodeTuiCommandExecuteData     `json:"data"`
	Durable  *OpenCodeTuiCommandExecuteDurable `json:"durable,omitempty"`
	ID       string                            `json:"id"`
	Location OpenCodeLocationRef               `json:"location,omitempty"`
	Metadata map[string]any                    `json:"metadata,omitempty"`
	Type     string                            `json:"type"`
}

type OpenCodeTuiPromptAppendData struct {
	Text string `json:"text"`
}

type OpenCodeTuiPromptAppendDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeTuiPromptAppend struct {
	Data     OpenCodeTuiPromptAppendData     `json:"data"`
	Durable  *OpenCodeTuiPromptAppendDurable `json:"durable,omitempty"`
	ID       string                          `json:"id"`
	Location OpenCodeLocationRef             `json:"location,omitempty"`
	Metadata map[string]any                  `json:"metadata,omitempty"`
	Type     string                          `json:"type"`
}

type OpenCodeTuiSessionSelectData struct {
	SessionID string `json:"sessionID"`
}

type OpenCodeTuiSessionSelectDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeTuiSessionSelect struct {
	Data     OpenCodeTuiSessionSelectData     `json:"data"`
	Durable  *OpenCodeTuiSessionSelectDurable `json:"durable,omitempty"`
	ID       string                           `json:"id"`
	Location OpenCodeLocationRef              `json:"location,omitempty"`
	Metadata map[string]any                   `json:"metadata,omitempty"`
	Type     string                           `json:"type"`
}

type OpenCodeTuiToastShowData struct {
	Duration int64  `json:"duration,omitempty"`
	Message  string `json:"message"`
	Title    string `json:"title,omitempty"`
	Variant  string `json:"variant"`
}

type OpenCodeTuiToastShowDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeTuiToastShow struct {
	Data     OpenCodeTuiToastShowData     `json:"data"`
	Durable  *OpenCodeTuiToastShowDurable `json:"durable,omitempty"`
	ID       string                       `json:"id"`
	Location OpenCodeLocationRef          `json:"location,omitempty"`
	Metadata map[string]any               `json:"metadata,omitempty"`
	Type     string                       `json:"type"`
}

type OpenCodeUnauthorizedError struct {
	Tag     string `json:"_tag"`
	Message string `json:"message"`
}

type OpenCodeUnknownErrorData struct {
	Message string `json:"message"`
	Ref     string `json:"ref,omitempty"`
}

type OpenCodeUnknownError struct {
	Data OpenCodeUnknownErrorData `json:"data"`
	Name string                   `json:"name"`
}

type OpenCodeUnknownError1 struct {
	Tag     string `json:"_tag"`
	Message string `json:"message"`
	Ref     string `json:"ref,omitempty"`
}

type OpenCodeUserMessageModel struct {
	ModelID    string `json:"modelID"`
	ProviderID string `json:"providerID"`
	Variant    string `json:"variant,omitempty"`
}

type OpenCodeUserMessageSummary struct {
	Body  string                     `json:"body,omitempty"`
	Diffs []OpenCodeSnapshotFileDiff `json:"diffs"`
	Title string                     `json:"title,omitempty"`
}

type OpenCodeUserMessageTime struct {
	Created float64 `json:"created"`
}

type OpenCodeUserMessage struct {
	Agent     string                      `json:"agent"`
	Format    OpenCodeOutputFormat        `json:"format,omitempty"`
	ID        string                      `json:"id"`
	Model     OpenCodeUserMessageModel    `json:"model"`
	Role      string                      `json:"role"`
	SessionID string                      `json:"sessionID"`
	Summary   *OpenCodeUserMessageSummary `json:"summary,omitempty"`
	System    string                      `json:"system,omitempty"`
	Time      OpenCodeUserMessageTime     `json:"time"`
	Tools     map[string]bool             `json:"tools,omitempty"`
}

type OpenCodeV2Event any

type OpenCodeV2EventStream string

type OpenCodeVcsApplyErrorData struct {
	Message string `json:"message"`
	Reason  string `json:"reason"`
}

type OpenCodeVcsApplyError struct {
	Data OpenCodeVcsApplyErrorData `json:"data"`
	Name string                    `json:"name"`
}

type OpenCodeVcsBranchUpdatedData struct {
	Branch string `json:"branch,omitempty"`
}

type OpenCodeVcsBranchUpdatedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeVcsBranchUpdated struct {
	Data     OpenCodeVcsBranchUpdatedData     `json:"data"`
	Durable  *OpenCodeVcsBranchUpdatedDurable `json:"durable,omitempty"`
	ID       string                           `json:"id"`
	Location OpenCodeLocationRef              `json:"location,omitempty"`
	Metadata map[string]any                   `json:"metadata,omitempty"`
	Type     string                           `json:"type"`
}

type OpenCodeVcsFileDiff struct {
	Additions float64 `json:"additions"`
	Deletions float64 `json:"deletions"`
	File      string  `json:"file"`
	Patch     string  `json:"patch,omitempty"`
	Status    string  `json:"status,omitempty"`
}

type OpenCodeVcsFileStatus struct {
	Additions float64 `json:"additions"`
	Deletions float64 `json:"deletions"`
	File      string  `json:"file"`
	Status    string  `json:"status"`
}

type OpenCodeVcsInfo struct {
	Branch        string `json:"branch,omitempty"`
	DefaultBranch string `json:"default_branch,omitempty"`
}

type OpenCodeWellKnownAuth struct {
	Key   string `json:"key"`
	Token string `json:"token"`
	Type  string `json:"type"`
}

type OpenCodeWorkspace struct {
	Branch    any    `json:"branch,omitempty"`
	Directory any    `json:"directory,omitempty"`
	Extra     any    `json:"extra,omitempty"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	ProjectID string `json:"projectID"`
	TimeUsed  any    `json:"timeUsed"`
	Type      string `json:"type"`
}

type OpenCodeWorkspaceCreateErrorData struct {
	Message string `json:"message"`
}

type OpenCodeWorkspaceCreateError struct {
	Data OpenCodeWorkspaceCreateErrorData `json:"data"`
	Name string                           `json:"name"`
}

type OpenCodeWorkspaceEventConnectionStatus struct {
	Status      string `json:"status"`
	WorkspaceID string `json:"workspaceID"`
}

type OpenCodeWorkspaceFailedData struct {
	Message string `json:"message"`
}

type OpenCodeWorkspaceFailedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeWorkspaceFailed struct {
	Data     OpenCodeWorkspaceFailedData     `json:"data"`
	Durable  *OpenCodeWorkspaceFailedDurable `json:"durable,omitempty"`
	ID       string                          `json:"id"`
	Location OpenCodeLocationRef             `json:"location,omitempty"`
	Metadata map[string]any                  `json:"metadata,omitempty"`
	Type     string                          `json:"type"`
}

type OpenCodeWorkspaceReadyData struct {
	Name string `json:"name"`
}

type OpenCodeWorkspaceReadyDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeWorkspaceReady struct {
	Data     OpenCodeWorkspaceReadyData     `json:"data"`
	Durable  *OpenCodeWorkspaceReadyDurable `json:"durable,omitempty"`
	ID       string                         `json:"id"`
	Location OpenCodeLocationRef            `json:"location,omitempty"`
	Metadata map[string]any                 `json:"metadata,omitempty"`
	Type     string                         `json:"type"`
}

type OpenCodeWorkspaceStatusData struct {
	Status      string `json:"status"`
	WorkspaceID string `json:"workspaceID"`
}

type OpenCodeWorkspaceStatusDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeWorkspaceStatus struct {
	Data     OpenCodeWorkspaceStatusData     `json:"data"`
	Durable  *OpenCodeWorkspaceStatusDurable `json:"durable,omitempty"`
	ID       string                          `json:"id"`
	Location OpenCodeLocationRef             `json:"location,omitempty"`
	Metadata map[string]any                  `json:"metadata,omitempty"`
	Type     string                          `json:"type"`
}

type OpenCodeWorkspaceWarpErrorData struct {
	Message string `json:"message"`
}

type OpenCodeWorkspaceWarpError struct {
	Data OpenCodeWorkspaceWarpErrorData `json:"data"`
	Name string                         `json:"name"`
}

type OpenCodeWorktree struct {
	Branch    string `json:"branch,omitempty"`
	Directory string `json:"directory"`
	Name      string `json:"name"`
}

type OpenCodeWorktreeCreateInput struct {
	Name         string `json:"name,omitempty"`
	StartCommand string `json:"startCommand,omitempty"`
}

type OpenCodeWorktreeErrorData struct {
	Message string `json:"message"`
}

type OpenCodeWorktreeError struct {
	Data OpenCodeWorktreeErrorData `json:"data"`
	Name string                    `json:"name"`
}

type OpenCodeWorktreeFailedData struct {
	Message string `json:"message"`
}

type OpenCodeWorktreeFailedDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeWorktreeFailed struct {
	Data     OpenCodeWorktreeFailedData     `json:"data"`
	Durable  *OpenCodeWorktreeFailedDurable `json:"durable,omitempty"`
	ID       string                         `json:"id"`
	Location OpenCodeLocationRef            `json:"location,omitempty"`
	Metadata map[string]any                 `json:"metadata,omitempty"`
	Type     string                         `json:"type"`
}

type OpenCodeWorktreeReadyData struct {
	Branch string `json:"branch,omitempty"`
	Name   string `json:"name"`
}

type OpenCodeWorktreeReadyDurable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeWorktreeReady struct {
	Data     OpenCodeWorktreeReadyData     `json:"data"`
	Durable  *OpenCodeWorktreeReadyDurable `json:"durable,omitempty"`
	ID       string                        `json:"id"`
	Location OpenCodeLocationRef           `json:"location,omitempty"`
	Metadata map[string]any                `json:"metadata,omitempty"`
	Type     string                        `json:"type"`
}

type OpenCodeWorktreeRemoveInput struct {
	Directory string `json:"directory"`
}

type OpenCodeWorktreeResetInput struct {
	Directory string `json:"directory"`
}

type OpenCodeEffectHttpApiErrorBadRequest struct {
	Tag string `json:"_tag"`
}

type OpenCodeEffectHttpApiErrorForbidden struct {
	Tag string `json:"_tag"`
}

type OpenCodeEffectHttpApiErrorInternalServerError struct {
	Tag string `json:"_tag"`
}

type OpenCodeQuestionRejected2Data struct {
	RequestID string `json:"requestID"`
	SessionID string `json:"sessionID"`
}

type OpenCodeQuestionRejected2Durable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeQuestionRejected2 struct {
	Data     OpenCodeQuestionRejected2Data     `json:"data"`
	Durable  *OpenCodeQuestionRejected2Durable `json:"durable,omitempty"`
	ID       string                            `json:"id"`
	Location OpenCodeLocationRef               `json:"location,omitempty"`
	Metadata map[string]any                    `json:"metadata,omitempty"`
	Type     string                            `json:"type"`
}

type OpenCodeQuestionReplied2Data struct {
	Answers   []OpenCodeQuestionAnswer `json:"answers"`
	RequestID string                   `json:"requestID"`
	SessionID string                   `json:"sessionID"`
}

type OpenCodeQuestionReplied2Durable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeQuestionReplied2 struct {
	Data     OpenCodeQuestionReplied2Data     `json:"data"`
	Durable  *OpenCodeQuestionReplied2Durable `json:"durable,omitempty"`
	ID       string                           `json:"id"`
	Location OpenCodeLocationRef              `json:"location,omitempty"`
	Metadata map[string]any                   `json:"metadata,omitempty"`
	Type     string                           `json:"type"`
}

type OpenCodeSessionStatus2Data struct {
	SessionID string                `json:"sessionID"`
	Status    OpenCodeSessionStatus `json:"status"`
}

type OpenCodeSessionStatus2Durable struct {
	AggregateID string `json:"aggregateID"`
	Seq         int64  `json:"seq"`
	Version     int64  `json:"version"`
}

type OpenCodeSessionStatus2 struct {
	Data     OpenCodeSessionStatus2Data     `json:"data"`
	Durable  *OpenCodeSessionStatus2Durable `json:"durable,omitempty"`
	ID       string                         `json:"id"`
	Location OpenCodeLocationRef            `json:"location,omitempty"`
	Metadata map[string]any                 `json:"metadata,omitempty"`
	Type     string                         `json:"type"`
}

type OpenCodeAppAgentsResponse []OpenCodeAgent

type OpenCodeV2AgentListResponse struct {
	Data     []OpenCodeAgentV2Info `json:"data"`
	Location OpenCodeLocationInfo  `json:"location"`
}

type OpenCodeV2CommandListResponse struct {
	Data     []OpenCodeCommandV2Info `json:"data"`
	Location OpenCodeLocationInfo    `json:"location"`
}

type OpenCodeV2CredentialUpdateRequest struct {
	Label string `json:"label"`
}

type OpenCodeV2FsFindResponse struct {
	Data     []OpenCodeFileSystemEntry `json:"data"`
	Location OpenCodeLocationInfo      `json:"location"`
}

type OpenCodeV2FsListResponse struct {
	Data     []OpenCodeFileSystemEntry `json:"data"`
	Location OpenCodeLocationInfo      `json:"location"`
}

type OpenCodeV2HealthGetResponse struct {
	Healthy bool `json:"healthy"`
}

type OpenCodeV2IntegrationListResponse struct {
	Data     []OpenCodeIntegrationInfo `json:"data"`
	Location OpenCodeLocationInfo      `json:"location"`
}

type OpenCodeV2IntegrationAttemptStatusResponse struct {
	Data     OpenCodeIntegrationAttemptStatus `json:"data"`
	Location OpenCodeLocationInfo             `json:"location"`
}

type OpenCodeV2IntegrationAttemptCompleteRequest struct {
	Code string `json:"code,omitempty"`
}

type OpenCodeV2IntegrationGetResponse struct {
	Data     OpenCodeIntegrationInfo `json:"data"`
	Location OpenCodeLocationInfo    `json:"location"`
}

type OpenCodeV2IntegrationConnectKeyRequest struct {
	Key   string `json:"key"`
	Label string `json:"label,omitempty"`
}

type OpenCodeV2IntegrationConnectOauthRequest struct {
	Inputs   map[string]string `json:"inputs"`
	Label    string            `json:"label,omitempty"`
	MethodID string            `json:"methodID"`
}

type OpenCodeV2IntegrationConnectOauthResponse struct {
	Data     OpenCodeIntegrationAttempt `json:"data"`
	Location OpenCodeLocationInfo       `json:"location"`
}

type OpenCodeV2LocationGetResponse = OpenCodeLocationInfo

type OpenCodeV2ModelListResponse struct {
	Data     []OpenCodeModelV2Info `json:"data"`
	Location OpenCodeLocationInfo  `json:"location"`
}

type OpenCodeV2PermissionRequestListResponse struct {
	Data     []OpenCodePermissionV2Request `json:"data"`
	Location OpenCodeLocationInfo          `json:"location"`
}

type OpenCodeV2PermissionSavedListResponse struct {
	Data []OpenCodePermissionSavedInfo `json:"data"`
}

type OpenCodeV2ProviderListResponse struct {
	Data     []OpenCodeProviderV2Info `json:"data"`
	Location OpenCodeLocationInfo     `json:"location"`
}

type OpenCodeV2ProviderGetResponse struct {
	Data     OpenCodeProviderV2Info `json:"data"`
	Location OpenCodeLocationInfo   `json:"location"`
}

type OpenCodeV2PtyListResponse struct {
	Data     []OpenCodePty        `json:"data"`
	Location OpenCodeLocationInfo `json:"location"`
}

type OpenCodeV2PtyCreateRequest struct {
	Args    []string          `json:"args,omitempty"`
	Command string            `json:"command,omitempty"`
	Cwd     string            `json:"cwd,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Title   string            `json:"title,omitempty"`
}

type OpenCodeV2PtyCreateResponse struct {
	Data     OpenCodePty          `json:"data"`
	Location OpenCodeLocationInfo `json:"location"`
}

type OpenCodeV2PtyGetResponse struct {
	Data     OpenCodePty          `json:"data"`
	Location OpenCodeLocationInfo `json:"location"`
}

type OpenCodeV2PtyUpdateRequestSize struct {
	Cols int64 `json:"cols"`
	Rows int64 `json:"rows"`
}

type OpenCodeV2PtyUpdateRequest struct {
	Size  *OpenCodeV2PtyUpdateRequestSize `json:"size,omitempty"`
	Title string                          `json:"title,omitempty"`
}

type OpenCodeV2PtyUpdateResponse struct {
	Data     OpenCodePty          `json:"data"`
	Location OpenCodeLocationInfo `json:"location"`
}

type OpenCodeV2PtyConnectResponse bool

type OpenCodeV2PtyConnectTokenResponse struct {
	Data     OpenCodePtyTicketConnectToken `json:"data"`
	Location OpenCodeLocationInfo          `json:"location"`
}

type OpenCodeV2QuestionRequestListResponse struct {
	Data     []OpenCodeQuestionV2Request `json:"data"`
	Location OpenCodeLocationInfo        `json:"location"`
}

type OpenCodeV2ReferenceListResponse struct {
	Data     []OpenCodeReferenceInfo `json:"data"`
	Location OpenCodeLocationInfo    `json:"location"`
}

type OpenCodeV2SessionListResponse = OpenCodeSessionsResponse

type OpenCodeV2SessionCreateRequest struct {
	Agent    string              `json:"agent,omitempty"`
	ID       string              `json:"id,omitempty"`
	Location OpenCodeLocationRef `json:"location,omitempty"`
	Model    OpenCodeModelRef    `json:"model,omitempty"`
}

type OpenCodeV2SessionCreateResponse struct {
	Data OpenCodeSessionV2Info `json:"data"`
}

type OpenCodeV2SessionActiveResponse struct {
	Data map[string]any `json:"data"`
}

type OpenCodeV2SessionGetResponse struct {
	Data OpenCodeSessionV2Info `json:"data"`
}

type OpenCodeV2SessionSwitchAgentRequest struct {
	Agent string `json:"agent"`
}

type OpenCodeV2SessionContextResponse struct {
	Data []OpenCodeSessionMessage `json:"data"`
}

type OpenCodeV2SessionHistoryResponse = OpenCodeSessionHistory

type OpenCodeV2SessionMessagesResponse = OpenCodeSessionMessagesResponse

type OpenCodeV2SessionMessageResponse struct {
	Data OpenCodeSessionMessage `json:"data"`
}

type OpenCodeV2SessionSwitchModelRequest struct {
	Model OpenCodeModelRef `json:"model"`
}

type OpenCodeV2SessionPermissionListResponse struct {
	Data []OpenCodePermissionV2Request `json:"data"`
}

type OpenCodeV2SessionPermissionCreateRequest struct {
	Action    string                     `json:"action"`
	Agent     string                     `json:"agent,omitempty"`
	ID        string                     `json:"id,omitempty"`
	Metadata  map[string]any             `json:"metadata,omitempty"`
	Resources []string                   `json:"resources"`
	Save      []string                   `json:"save,omitempty"`
	Source    OpenCodePermissionV2Source `json:"source,omitempty"`
}

type OpenCodeV2SessionPermissionCreateResponseData struct {
	Effect OpenCodePermissionV2Effect `json:"effect"`
	ID     string                     `json:"id"`
}

type OpenCodeV2SessionPermissionCreateResponse struct {
	Data OpenCodeV2SessionPermissionCreateResponseData `json:"data"`
}

type OpenCodeV2SessionPermissionGetResponse struct {
	Data OpenCodePermissionV2Request `json:"data"`
}

type OpenCodeV2SessionPermissionReplyRequest struct {
	Message string                    `json:"message,omitempty"`
	Reply   OpenCodePermissionV2Reply `json:"reply"`
}

type OpenCodeV2SessionPromptRequest struct {
	Delivery string              `json:"delivery,omitempty"`
	ID       string              `json:"id,omitempty"`
	Prompt   OpenCodePromptInput `json:"prompt"`
	Resume   bool                `json:"resume,omitempty"`
}

type OpenCodeV2SessionPromptResponse struct {
	Data OpenCodeSessionInputAdmitted `json:"data"`
}

type OpenCodeV2SessionQuestionListResponse struct {
	Data []OpenCodeQuestionV2Request `json:"data"`
}

type OpenCodeV2SessionQuestionReplyRequest = OpenCodeQuestionV2Reply

type OpenCodeV2SessionRevertStageRequest struct {
	Files     bool   `json:"files,omitempty"`
	MessageID string `json:"messageID"`
}

type OpenCodeV2SessionRevertStageResponse struct {
	Data OpenCodeRevertState `json:"data"`
}

type OpenCodeV2SkillListResponse struct {
	Data     []OpenCodeSkillV2Info `json:"data"`
	Location OpenCodeLocationInfo  `json:"location"`
}

type OpenCodeAuthSetRequest = OpenCodeAuth

type OpenCodeAuthSetResponse bool

type OpenCodeAuthRemoveResponse bool

type OpenCodeCommandListResponse []OpenCodeCommand

type OpenCodeConfigGetResponse = OpenCodeConfig

type OpenCodeConfigUpdateRequest = OpenCodeConfig

type OpenCodeConfigUpdateResponse = OpenCodeConfig

type OpenCodeConfigProvidersResponse struct {
	Default   map[string]string  `json:"default"`
	Providers []OpenCodeProvider `json:"providers"`
}

type OpenCodeExperimentalCapabilitiesGetResponse = OpenCodeExperimentalCapabilities

type OpenCodeExperimentalConsoleGetResponse = OpenCodeConsoleState

type OpenCodeExperimentalConsoleListOrgsResponseOrgsItem struct {
	AccountEmail string `json:"accountEmail"`
	AccountID    string `json:"accountID"`
	AccountUrl   string `json:"accountUrl"`
	Active       bool   `json:"active"`
	OrgID        string `json:"orgID"`
	OrgName      string `json:"orgName"`
}

type OpenCodeExperimentalConsoleListOrgsResponse struct {
	Orgs []OpenCodeExperimentalConsoleListOrgsResponseOrgsItem `json:"orgs"`
}

type OpenCodeExperimentalConsoleSwitchOrgRequest struct {
	AccountID string `json:"accountID"`
	OrgID     string `json:"orgID"`
}

type OpenCodeExperimentalConsoleSwitchOrgResponse bool

type OpenCodeExperimentalControlPlaneMoveSessionRequest struct {
	Destination OpenCodeMoveSessionDestination `json:"destination"`
	MoveChanges bool                           `json:"moveChanges,omitempty"`
	SessionID   string                         `json:"sessionID"`
}

type OpenCodeV2ProjectCopyCreateRequest struct {
	Directory string `json:"directory"`
	Name      string `json:"name,omitempty"`
	Strategy  string `json:"strategy"`
}

type OpenCodeV2ProjectCopyCreateResponse = OpenCodeProjectCopyCopy

type OpenCodeV2ProjectCopyRemoveRequest struct {
	Directory string `json:"directory"`
	Force     bool   `json:"force"`
}

type OpenCodeExperimentalProjectCopyGenerateNameRequest struct {
	Context string `json:"context,omitempty"`
}

type OpenCodeExperimentalProjectCopyGenerateNameResponse struct {
	Name string `json:"name"`
}

type OpenCodeExperimentalResourceListResponse struct {
}

type OpenCodeExperimentalSessionListResponse []OpenCodeGlobalSession

type OpenCodeExperimentalSessionBackgroundResponse bool

type OpenCodeToolListResponse = OpenCodeToolList

type OpenCodeToolIdsResponse = OpenCodeToolIDs

type OpenCodeExperimentalWorkspaceListResponse []OpenCodeWorkspace

type OpenCodeExperimentalWorkspaceCreateRequest struct {
	Branch any    `json:"branch,omitempty"`
	Extra  any    `json:"extra,omitempty"`
	ID     string `json:"id,omitempty"`
	Type   string `json:"type"`
}

type OpenCodeExperimentalWorkspaceCreateResponse = OpenCodeWorkspace

type OpenCodeExperimentalWorkspaceAdapterListResponse []struct {
	Description string `json:"description"`
	Name        string `json:"name"`
	Type        string `json:"type"`
}

type OpenCodeExperimentalWorkspaceStatusResponse []OpenCodeWorkspaceEventConnectionStatus

type OpenCodeExperimentalWorkspaceWarpRequest struct {
	CopyChanges bool   `json:"copyChanges,omitempty"`
	ID          any    `json:"id"`
	SessionID   string `json:"sessionID"`
}

type OpenCodeExperimentalWorkspaceRemoveResponse = OpenCodeWorkspace

type OpenCodeWorktreeListResponse []string

type OpenCodeWorktreeCreateRequest = OpenCodeWorktreeCreateInput

type OpenCodeWorktreeCreateResponse = OpenCodeWorktree

type OpenCodeWorktreeRemoveRequest = OpenCodeWorktreeRemoveInput

type OpenCodeWorktreeRemoveResponse bool

type OpenCodeWorktreeResetRequest = OpenCodeWorktreeResetInput

type OpenCodeWorktreeResetResponse bool

type OpenCodeFileListResponse []OpenCodeFileNode

type OpenCodeFileReadResponse = OpenCodeFileContent

type OpenCodeFileStatusResponse []OpenCodeFile

type OpenCodeFindTextResponse []struct {
	AbsoluteOffset int64 `json:"absolute_offset"`
	LineNumber     int64 `json:"line_number"`
	Lines          struct {
		Text string `json:"text"`
	} `json:"lines"`
	Path struct {
		Text string `json:"text"`
	} `json:"path"`
	Submatches []struct {
		End   int64 `json:"end"`
		Match struct {
			Text string `json:"text"`
		} `json:"match"`
		Start int64 `json:"start"`
	} `json:"submatches"`
}

type OpenCodeFindFilesResponse []string

type OpenCodeFindSymbolsResponse []OpenCodeSymbol

type OpenCodeFormatterStatusResponse []OpenCodeFormatterStatus

type OpenCodeGlobalConfigGetResponse = OpenCodeConfig

type OpenCodeGlobalConfigUpdateRequest = OpenCodeConfig

type OpenCodeGlobalConfigUpdateResponse = OpenCodeConfig

type OpenCodeGlobalDisposeResponse bool

type OpenCodeGlobalHealthResponse struct {
	Healthy bool   `json:"healthy"`
	Version string `json:"version"`
}

type OpenCodeGlobalUpgradeRequest struct {
	Target string `json:"target,omitempty"`
}

type OpenCodeGlobalUpgradeResponse any

type OpenCodeInstanceDisposeResponse bool

type OpenCodeAppLogRequest struct {
	Extra   map[string]any `json:"extra,omitempty"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Service string         `json:"service"`
}

type OpenCodeAppLogResponse bool

type OpenCodeLspStatusResponse []OpenCodeLSPStatus

type OpenCodeMcpStatusResponse struct {
}

type OpenCodeMcpAddRequest struct {
	Config any    `json:"config"`
	Name   string `json:"name"`
}

type OpenCodeMcpAddResponse struct {
}

type OpenCodeMcpAuthStartResponse struct {
	AuthorizationUrl string `json:"authorizationUrl"`
	OauthState       string `json:"oauthState"`
}

type OpenCodeMcpAuthRemoveResponse struct {
	Success bool `json:"success"`
}

type OpenCodeMcpAuthAuthenticateResponse = OpenCodeMCPStatus

type OpenCodeMcpAuthCallbackRequest struct {
	Code string `json:"code"`
}

type OpenCodeMcpAuthCallbackResponse = OpenCodeMCPStatus

type OpenCodeMcpConnectResponse bool

type OpenCodeMcpDisconnectResponse bool

type OpenCodePathGetResponse = OpenCodePath

type OpenCodePermissionListResponse []OpenCodePermissionRequest

type OpenCodePermissionReplyRequest struct {
	Message string `json:"message,omitempty"`
	Reply   string `json:"reply"`
}

type OpenCodePermissionReplyResponse bool

type OpenCodeProjectListResponse []OpenCodeProject

type OpenCodeProjectCurrentResponse = OpenCodeProject

type OpenCodeProjectInitGitResponse = OpenCodeProject

type OpenCodeProjectUpdateRequest struct {
	Commands OpenCodeProjectCommands `json:"commands,omitempty"`
	Icon     OpenCodeProjectIcon     `json:"icon,omitempty"`
	Name     string                  `json:"name,omitempty"`
}

type OpenCodeProjectUpdateResponse = OpenCodeProject

type OpenCodeProjectDirectoriesResponse = OpenCodeProjectDirectories

type OpenCodeProviderListResponse struct {
	All       []OpenCodeProvider `json:"all"`
	Connected []string           `json:"connected"`
	Default   map[string]string  `json:"default"`
}

type OpenCodeProviderAuthResponse struct {
}

type OpenCodeProviderOauthAuthorizeRequest struct {
	Inputs map[string]string `json:"inputs,omitempty"`
	Method float64           `json:"method"`
}

type OpenCodeProviderOauthAuthorizeResponse = OpenCodeProviderAuthAuthorization

type OpenCodeProviderOauthCallbackRequest struct {
	Code   string  `json:"code,omitempty"`
	Method float64 `json:"method"`
}

type OpenCodeProviderOauthCallbackResponse bool

type OpenCodePtyListResponse []OpenCodePty

type OpenCodePtyCreateRequest struct {
	Args    []string          `json:"args,omitempty"`
	Command string            `json:"command,omitempty"`
	Cwd     string            `json:"cwd,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Title   string            `json:"title,omitempty"`
}

type OpenCodePtyCreateResponse = OpenCodePty

type OpenCodePtyShellsResponse []struct {
	Acceptable bool   `json:"acceptable"`
	Name       string `json:"name"`
	Path       string `json:"path"`
}

type OpenCodePtyGetResponse = OpenCodePty

type OpenCodePtyUpdateRequestSize struct {
	Cols int64 `json:"cols"`
	Rows int64 `json:"rows"`
}

type OpenCodePtyUpdateRequest struct {
	Size  *OpenCodePtyUpdateRequestSize `json:"size,omitempty"`
	Title string                        `json:"title,omitempty"`
}

type OpenCodePtyUpdateResponse = OpenCodePty

type OpenCodePtyRemoveResponse bool

type OpenCodePtyConnectResponse bool

type OpenCodePtyConnectTokenResponse = OpenCodePtyTicketConnectToken

type OpenCodeQuestionListResponse []OpenCodeQuestionRequest

type OpenCodeQuestionRejectResponse bool

type OpenCodeQuestionReplyRequest struct {
	Answers []OpenCodeQuestionAnswer `json:"answers"`
}

type OpenCodeQuestionReplyResponse bool

type OpenCodeSessionListResponse []OpenCodeSession

type OpenCodeSessionCreateRequestModel struct {
	ID         string `json:"id"`
	ProviderID string `json:"providerID"`
	Variant    string `json:"variant,omitempty"`
}

type OpenCodeSessionCreateRequest struct {
	Agent       string                             `json:"agent,omitempty"`
	Metadata    map[string]any                     `json:"metadata,omitempty"`
	Model       *OpenCodeSessionCreateRequestModel `json:"model,omitempty"`
	ParentID    string                             `json:"parentID,omitempty"`
	Permission  OpenCodePermissionRuleset          `json:"permission,omitempty"`
	Title       string                             `json:"title,omitempty"`
	WorkspaceID string                             `json:"workspaceID,omitempty"`
}

type OpenCodeSessionCreateResponse = OpenCodeSession

type OpenCodeSessionStatusResponse struct {
}

type OpenCodeSessionGetResponse = OpenCodeSession

type OpenCodeSessionUpdateRequestTime struct {
	Archived float64 `json:"archived,omitempty"`
}

type OpenCodeSessionUpdateRequest struct {
	Metadata   map[string]any                    `json:"metadata,omitempty"`
	Permission OpenCodePermissionRuleset         `json:"permission,omitempty"`
	Time       *OpenCodeSessionUpdateRequestTime `json:"time,omitempty"`
	Title      string                            `json:"title,omitempty"`
}

type OpenCodeSessionUpdateResponse = OpenCodeSession

type OpenCodeSessionDeleteResponse bool

type OpenCodeSessionAbortResponse bool

type OpenCodeSessionChildrenResponse []OpenCodeSession

type OpenCodeSessionCommandRequestPartsItem struct {
	Filename string                 `json:"filename,omitempty"`
	ID       string                 `json:"id,omitempty"`
	Mime     string                 `json:"mime"`
	Source   OpenCodeFilePartSource `json:"source,omitempty"`
	Type     string                 `json:"type"`
	Url      string                 `json:"url"`
}

type OpenCodeSessionCommandRequest struct {
	Agent     string                                   `json:"agent,omitempty"`
	Arguments string                                   `json:"arguments"`
	Command   string                                   `json:"command"`
	MessageID string                                   `json:"messageID,omitempty"`
	Model     string                                   `json:"model,omitempty"`
	Parts     []OpenCodeSessionCommandRequestPartsItem `json:"parts,omitempty"`
	Variant   string                                   `json:"variant,omitempty"`
}

type OpenCodeSessionCommandResponse struct {
	Info  OpenCodeAssistantMessage `json:"info"`
	Parts []OpenCodePart           `json:"parts"`
}

type OpenCodeSessionDiffResponse []OpenCodeSnapshotFileDiff

type OpenCodeSessionForkRequest struct {
	MessageID string `json:"messageID,omitempty"`
}

type OpenCodeSessionForkResponse = OpenCodeSession

type OpenCodeSessionInitRequest struct {
	MessageID  string `json:"messageID"`
	ModelID    string `json:"modelID"`
	ProviderID string `json:"providerID"`
}

type OpenCodeSessionInitResponse bool

type OpenCodeSessionMessagesResponse2 []struct {
	Info  OpenCodeMessage `json:"info"`
	Parts []OpenCodePart  `json:"parts"`
}

type OpenCodeSessionPromptRequestModel struct {
	ModelID    string `json:"modelID"`
	ProviderID string `json:"providerID"`
}

type OpenCodeSessionPromptRequest struct {
	Agent     string                             `json:"agent,omitempty"`
	Format    OpenCodeOutputFormat               `json:"format,omitempty"`
	MessageID string                             `json:"messageID,omitempty"`
	Model     *OpenCodeSessionPromptRequestModel `json:"model,omitempty"`
	NoReply   bool                               `json:"noReply,omitempty"`
	Parts     []any                              `json:"parts"`
	System    string                             `json:"system,omitempty"`
	Tools     map[string]bool                    `json:"tools,omitempty"`
	Variant   string                             `json:"variant,omitempty"`
}

type OpenCodeSessionPromptResponse struct {
	Info  OpenCodeAssistantMessage `json:"info"`
	Parts []OpenCodePart           `json:"parts"`
}

type OpenCodeSessionMessageResponse struct {
	Info  OpenCodeMessage `json:"info"`
	Parts []OpenCodePart  `json:"parts"`
}

type OpenCodeSessionDeleteMessageResponse bool

type OpenCodePartUpdateRequest = OpenCodePart

type OpenCodePartUpdateResponse = OpenCodePart

type OpenCodePartDeleteResponse bool

type OpenCodePermissionRespondRequest struct {
	Response string `json:"response"`
}

type OpenCodePermissionRespondResponse bool

type OpenCodeSessionPromptAsyncRequestModel struct {
	ModelID    string `json:"modelID"`
	ProviderID string `json:"providerID"`
}

type OpenCodeSessionPromptAsyncRequest struct {
	Agent     string                                  `json:"agent,omitempty"`
	Format    OpenCodeOutputFormat                    `json:"format,omitempty"`
	MessageID string                                  `json:"id,omitempty"`
	Model     *OpenCodeSessionPromptAsyncRequestModel `json:"model,omitempty"`
	NoReply   bool                                    `json:"noReply,omitempty"`
	Parts     []any                                   `json:"parts"`
	System    string                                  `json:"system,omitempty"`
	Tools     map[string]bool                         `json:"tools,omitempty"`
	Variant   string                                  `json:"variant,omitempty"`
}

type OpenCodeSessionRevertRequest struct {
	MessageID string `json:"messageID"`
	PartID    string `json:"partID,omitempty"`
}

type OpenCodeSessionRevertResponse = OpenCodeSession

type OpenCodeSessionShareResponse = OpenCodeSession

type OpenCodeSessionUnshareResponse = OpenCodeSession

type OpenCodeSessionShellRequestModel struct {
	ModelID    string `json:"modelID"`
	ProviderID string `json:"providerID"`
}

type OpenCodeSessionShellRequest struct {
	Agent     string                            `json:"agent"`
	Command   string                            `json:"command"`
	MessageID string                            `json:"messageID,omitempty"`
	Model     *OpenCodeSessionShellRequestModel `json:"model,omitempty"`
}

type OpenCodeSessionShellResponse struct {
	Info  OpenCodeMessage `json:"info"`
	Parts []OpenCodePart  `json:"parts"`
}

type OpenCodeSessionSummarizeRequest struct {
	Auto       bool   `json:"auto,omitempty"`
	ModelID    string `json:"modelID"`
	ProviderID string `json:"providerID"`
}

type OpenCodeSessionSummarizeResponse bool

type OpenCodeSessionTodoResponse []OpenCodeTodo

type OpenCodeSessionUnrevertResponse = OpenCodeSession

type OpenCodeAppSkillsResponse []struct {
	Content     string `json:"content"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location"`
	Name        string `json:"name"`
}

type OpenCodeSyncHistoryListRequest struct {
}

type OpenCodeSyncHistoryListResponse []struct {
	AggregateId string         `json:"aggregate_id"`
	Data        map[string]any `json:"data"`
	ID          string         `json:"id"`
	Seq         int64          `json:"seq"`
	Type        string         `json:"type"`
}

type OpenCodeSyncReplayRequestEventsItem struct {
	AggregateID string         `json:"aggregateID"`
	Data        map[string]any `json:"data"`
	ID          string         `json:"id"`
	Seq         int64          `json:"seq"`
	Type        string         `json:"type"`
}

type OpenCodeSyncReplayRequest struct {
	Directory string                                `json:"directory"`
	Events    []OpenCodeSyncReplayRequestEventsItem `json:"events"`
}

type OpenCodeSyncReplayResponse struct {
	SessionID string `json:"sessionID"`
}

type OpenCodeSyncStartResponse bool

type OpenCodeSyncStealRequest struct {
	SessionID string `json:"sessionID"`
}

type OpenCodeSyncStealResponse struct {
	SessionID string `json:"sessionID"`
}

type OpenCodeTuiAppendPromptRequest struct {
	Text string `json:"text"`
}

type OpenCodeTuiAppendPromptResponse bool

type OpenCodeTuiClearPromptResponse bool

type OpenCodeTuiControlNextResponse struct {
	Body any    `json:"body"`
	Path string `json:"path"`
}

type OpenCodeTuiControlResponseRequest any

type OpenCodeTuiControlResponseResponse bool

type OpenCodeTuiExecuteCommandRequest struct {
	Command string `json:"command"`
}

type OpenCodeTuiExecuteCommandResponse bool

type OpenCodeTuiOpenHelpResponse bool

type OpenCodeTuiOpenModelsResponse bool

type OpenCodeTuiOpenSessionsResponse bool

type OpenCodeTuiOpenThemesResponse bool

type OpenCodeTuiPublishRequest any

type OpenCodeTuiPublishResponse bool

type OpenCodeTuiSelectSessionRequest struct {
	SessionID string `json:"sessionID"`
}

type OpenCodeTuiSelectSessionResponse bool

type OpenCodeTuiShowToastRequest struct {
	Duration int64  `json:"duration,omitempty"`
	Message  string `json:"message"`
	Title    string `json:"title,omitempty"`
	Variant  string `json:"variant"`
}

type OpenCodeTuiShowToastResponse bool

type OpenCodeTuiSubmitPromptResponse bool

type OpenCodeVcsGetResponse = OpenCodeVcsInfo

type OpenCodeVcsApplyRequest struct {
	Patch string `json:"patch"`
}

type OpenCodeVcsApplyResponse struct {
	Applied bool `json:"applied"`
}

type OpenCodeVcsDiffResponse []OpenCodeVcsFileDiff

type OpenCodeVcsStatusResponse []OpenCodeVcsFileStatus
