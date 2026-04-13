classDiagram
  namespace github_com_zopdev_reqflow {
    class github_com_zopdev_reqflow_ExtractAPIMap {
      <<function>>
    }
    class github_com_zopdev_reqflow_isBindMethod {
      <<function>>
    }
    class github_com_zopdev_reqflow_isResponseMethod {
      <<function>>
    }
    class github_com_zopdev_reqflow_extractTypeName {
      <<function>>
    }
    class github_com_zopdev_reqflow_appendUnique {
      <<function>>
    }
    class github_com_zopdev_reqflow_ExtractCallGraph {
      <<function>>
    }
    class github_com_zopdev_reqflow_resolveNodeID {
      <<function>>
    }
    class github_com_zopdev_reqflow_ReqflowConfig {
      +Linter struct{VetRules string "yaml:\"vet_rules\""}
      +Parser struct{IgnorePackages string "yaml:\"ignore_packages\""; DomainNaming struct{ServiceMatch string "yaml:\"service_match\""; StoreMatch string "yaml:\"store_match\""; ModelMatch string "yaml:\"model_match\""} "yaml:\"domain_naming\""}
      +Thresholds struct{MaxCycles int "yaml:\"max_cycles\""; MaxOrphans int "yaml:\"max_orphans\""; MaxSecurityIssues int "yaml:\"max_security_issues\""}
      +ServiceRegex regexp.Regexp
      +StoreRegex regexp.Regexp
      +ModelRegex regexp.Regexp
    }
    class github_com_zopdev_reqflow_LoadConfig {
      <<function>>
    }
    class github_com_zopdev_reqflow_DetectCycles {
      <<function>>
    }
    class github_com_zopdev_reqflow_DataFlow {
      +Entry string
      +Path string
      +Route string
    }
    class github_com_zopdev_reqflow_ExtractDataFlows {
      <<function>>
    }
    class github_com_zopdev_reqflow_DetectConcurrency {
      <<function>>
    }
    class github_com_zopdev_reqflow_tagConcurrency {
      <<function>>
    }
    class github_com_zopdev_reqflow_LoadCoverageProfile {
      <<function>>
    }
    class github_com_zopdev_reqflow_TechDebt {
      +File string
      +Line int
      +Kind string
      +Comment string
      +NodeID string
    }
    class github_com_zopdev_reqflow_DetectTechDebt {
      <<function>>
    }
    class github_com_zopdev_reqflow_findClosestNode {
      <<function>>
    }
    class github_com_zopdev_reqflow_MissingConstructor {
      +StructName string
      +Package string
      +File string
      +Line int
    }
    class github_com_zopdev_reqflow_DetectMissingConstructors {
      <<function>>
    }
    class github_com_zopdev_reqflow_SecurityIssue {
      +File string
      +Line int
      +Kind string
      +Detail string
      +Severity string
    }
    class github_com_zopdev_reqflow_DetectSecurityIssues {
      <<function>>
    }
    class github_com_zopdev_reqflow_goModule {
      <<model>>
      +Path string
      +Version string
      +Indirect bool
      +Main bool
      +Dir string
      +GoMod string
      +Replace github.com/thzgajendra/reqflow.goModule
    }
    class github_com_zopdev_reqflow_ExtractDepTree {
      <<function>>
    }
    class github_com_zopdev_reqflow_commonPrefix {
      <<function>>
    }
    class github_com_zopdev_reqflow_ExtractEnvMap {
      <<function>>
    }
    class github_com_zopdev_reqflow_extractEnvCall {
      <<function>>
    }
    class github_com_zopdev_reqflow_extractDefault {
      <<function>>
    }
    class github_com_zopdev_reqflow_findEnclosingNode {
      <<function>>
    }
    class github_com_zopdev_reqflow_SwallowedError {
      +File string
      +Line int
      +FuncName string
      +CallExpr string
    }
    class github_com_zopdev_reqflow_DetectSwallowedErrors {
      <<function>>
    }
    class github_com_zopdev_reqflow_isErrorTuple {
      <<function>>
    }
    class github_com_zopdev_reqflow_ExtractGoModDeps {
      <<function>>
    }
    class github_com_zopdev_reqflow_extractEvents {
      <<function>>
    }
    class github_com_zopdev_reqflow_extractTopicName {
      <<function>>
    }
    class github_com_zopdev_reqflow_extractKafkaConfigTopic {
      <<function>>
    }
    class github_com_zopdev_reqflow_EvolutionSnapshot {
      +Ref string
      +NodeCount int
      +EdgeCount int
      +Packages int
      +KindCount map[github.com/thzgajendra/reqflow.NodeKind]int
      +Added string
      +Removed string
    }
    class github_com_zopdev_reqflow_ExtractEvolution {
      <<function>>
    }
    class github_com_zopdev_reqflow_buildSnapshot {
      <<function>>
    }
    class github_com_zopdev_reqflow_getNodeIDSet {
      <<function>>
    }
    class github_com_zopdev_reqflow_sanitizeRef {
      <<function>>
    }
    class github_com_zopdev_reqflow_applyFocus {
      <<function>>
    }
    class github_com_zopdev_reqflow_ExtractContributors {
      <<function>>
    }
    class github_com_zopdev_reqflow_authorCount {
      +Name string
      +Count int
    }
    class github_com_zopdev_reqflow_gitAuthors {
      <<function>>
    }
    class github_com_zopdev_reqflow_ExtractChurn {
      <<function>>
    }
    class github_com_zopdev_reqflow_gitCommitCount {
      <<function>>
    }
    class github_com_zopdev_reqflow_GetChurnSummary {
      <<function>>
    }
    class github_com_zopdev_reqflow_ParseChurnTotal {
      <<function>>
    }
    class github_com_zopdev_reqflow_Node {
      +ID string
      +Kind github.com/thzgajendra/reqflow.NodeKind
      +Name string
      +Package string
      +Fields github.com/thzgajendra/reqflow.Field
      +Methods string
      +File string
      +Line int
      +Meta map[string]string
    }
    class github_com_zopdev_reqflow_Field {
      +Name string
      +Type string
      +Tag string
    }
    class github_com_zopdev_reqflow_Edge {
      +From string
      +To string
      +Kind github.com/thzgajendra/reqflow.EdgeKind
    }
    class github_com_zopdev_reqflow_Graph {
      +Nodes map[string]github.com/thzgajendra/reqflow.Node
      +Edges github.com/thzgajendra/reqflow.Edge
      +Clusters map[string]string
      +Meta map[string]string
      +AddNode()
      +AddEdge()
      +PrefixNodes()
    }
    class github_com_zopdev_reqflow_NewGraph {
      <<function>>
    }
    class github_com_zopdev_reqflow_ExtractInfraTopo {
      <<function>>
    }
    class github_com_zopdev_reqflow_parseDockerfile {
      <<function>>
    }
    class github_com_zopdev_reqflow_dockerComposeService {
      <<service>>
      +Image string
      +Build any
      +Ports string
      +DependsOn any
      +Networks string
      +Volumes string
      +Command string
      +Entrypoint string
    }
    class github_com_zopdev_reqflow_dockerComposeFile {
      +Services map[string]github.com/thzgajendra/reqflow.dockerComposeService
    }
    class github_com_zopdev_reqflow_parseDockerCompose {
      <<function>>
    }
    class github_com_zopdev_reqflow_extractDependsOn {
      <<function>>
    }
    class github_com_zopdev_reqflow_k8sManifest {
      +APIVersion string
      +Kind string
      +Metadata struct{Name string "yaml:\"name\""; Namespace string "yaml:\"namespace\""}
      +Spec struct{Template struct{Spec struct{Containers struct{Name string "yaml:\"name\""; Image string "yaml:\"image\""; Ports struct{ContainerPort int "yaml:\"containerPort\""} "yaml:\"ports\""} "yaml:\"containers\""} "yaml:\"spec\""} "yaml:\"template\""; ServicePorts struct{Port int "yaml:\"port\""; TargetPort int "yaml:\"targetPort\""; Protocol string "yaml:\"protocol\""} "yaml:\"ports\""; Selector map[string]string "yaml:\"selector\""}
    }
    class github_com_zopdev_reqflow_isK8sManifest {
      <<function>>
    }
    class github_com_zopdev_reqflow_parseK8sManifest {
      <<function>>
    }
    class github_com_zopdev_reqflow_sanitizePath {
      <<function>>
    }
    class github_com_zopdev_reqflow_NodeMetrics {
      +ID string
      +Name string
      +Kind github.com/thzgajendra/reqflow.NodeKind
      +FanIn int
      +FanOut int
      +Methods int
      +Package string
    }
    class github_com_zopdev_reqflow_ComputeMetrics {
      <<function>>
    }
    class github_com_zopdev_reqflow_ExtractMiddleware {
      <<function>>
    }
    class github_com_zopdev_reqflow_ExtractGRPC {
      <<function>>
    }
    class github_com_zopdev_reqflow_OTLPExport {
      <<model>>
      +ResourceSpans github.com/thzgajendra/reqflow.resourceSpan
    }
    class github_com_zopdev_reqflow_resourceSpan {
      <<model>>
      +Resource github.com/thzgajendra/reqflow.resource
      +ScopeSpans github.com/thzgajendra/reqflow.scopeSpan
    }
    class github_com_zopdev_reqflow_resource {
      <<handler>>
      +Attributes github.com/thzgajendra/reqflow.attribute
    }
    class github_com_zopdev_reqflow_scopeSpan {
      <<model>>
      +Spans github.com/thzgajendra/reqflow.span
    }
    class github_com_zopdev_reqflow_span {
      <<model>>
      +Name string
      +Kind int
      +StartTimeUnixNano string
      +EndTimeUnixNano string
      +Attributes github.com/thzgajendra/reqflow.attribute
      +Status github.com/thzgajendra/reqflow.spanStatus
      +ParentSpanID string
      +SpanID string
      +TraceID string
    }
    class github_com_zopdev_reqflow_attribute {
      <<model>>
      +Key string
      +Value github.com/thzgajendra/reqflow.attributeValue
    }
    class github_com_zopdev_reqflow_attributeValue {
      <<model>>
      +StringValue string
      +IntValue string
    }
    class github_com_zopdev_reqflow_spanStatus {
      <<model>>
      +Code int
      +Message string
    }
    class github_com_zopdev_reqflow_spanMetrics {
      +OperationName string
      +ServiceName string
      +Durations float64
      +ErrorCount int
      +TotalCount int
    }
    class github_com_zopdev_reqflow_ExtractOtelTrace {
      <<function>>
    }
    class github_com_zopdev_reqflow_extractServiceName {
      <<function>>
    }
    class github_com_zopdev_reqflow_spanDurationMs {
      <<function>>
    }
    class github_com_zopdev_reqflow_matchSpanToNode {
      <<function>>
    }
    class github_com_zopdev_reqflow_average {
      <<function>>
    }
    class github_com_zopdev_reqflow_percentile {
      <<function>>
    }
    class github_com_zopdev_reqflow_ParseOptions {
      +Dir string
      +Filter string
      +Focus string
      +Config github.com/thzgajendra/reqflow.ReqflowConfig
      +APIMap bool
      +Heatmap bool
      +CallGraph bool
      +DataFlow bool
      +EnvMap bool
      +TableMap bool
      +DepTree bool
      +InfraTopo bool
      +Churn bool
      +Contributors bool
      +PRImpact string
      +Evolution string
      +Proto bool
      +ServiceMap bool
      +OtelTrace string
    }
    class github_com_zopdev_reqflow_Parse {
      <<function>>
    }
    class github_com_zopdev_reqflow_shouldIgnorePackage {
      <<function>>
    }
    class github_com_zopdev_reqflow_handleTypeSpec {
      <<function>>
    }
    class github_com_zopdev_reqflow_handleFuncDecl {
      <<function>>
    }
    class github_com_zopdev_reqflow_isHTTPHandler {
      <<function>>
    }
    class github_com_zopdev_reqflow_extractDependencies {
      <<function>>
    }
    class github_com_zopdev_reqflow_resolveInterfaces {
      <<function>>
    }
    class github_com_zopdev_reqflow_resolveDependencies {
      <<function>>
    }
    class github_com_zopdev_reqflow_getModulePath {
      <<function>>
    }
    class github_com_zopdev_reqflow_matchLayer {
      <<function>>
    }
    class github_com_zopdev_reqflow_ExtractPRImpact {
      <<function>>
    }
    class github_com_zopdev_reqflow_ExtractProto {
      <<function>>
    }
    class github_com_zopdev_reqflow_parseProtoFile {
      <<function>>
    }
    class github_com_zopdev_reqflow_resolveProtoMsgID {
      <<function>>
    }
    class github_com_zopdev_reqflow_crossReferenceGRPC {
      <<function>>
    }
    class github_com_zopdev_reqflow_extractRoutes {
      <<function>>
    }
    class github_com_zopdev_reqflow_findAndTagHandler {
      <<function>>
    }
    class github_com_zopdev_reqflow_Stitch {
      <<function>>
    }
    class github_com_zopdev_reqflow_StitchWithServiceMap {
      <<function>>
    }
    class github_com_zopdev_reqflow_detectCrossServiceEdges {
      <<function>>
    }
    class github_com_zopdev_reqflow_PrintSummary {
      <<function>>
    }
    class github_com_zopdev_reqflow_GetSummaryHTML {
      <<function>>
    }
    class github_com_zopdev_reqflow_ExtractTableMap {
      <<function>>
    }
    class github_com_zopdev_reqflow_extractTagValue {
      <<function>>
    }
    class github_com_zopdev_reqflow_toSnakeCase {
      <<function>>
    }
    class github_com_zopdev_reqflow_VSchema {
      <<model>>
      +Sharded bool
      +Tables map[string]github.com/thzgajendra/reqflow.VTable
    }
    class github_com_zopdev_reqflow_VTable {
      <<model>>
      +ColumnVindexes struct{Column string "json:\"column\""; Name string "json:\"name\""}
    }
    class github_com_zopdev_reqflow_parseVitessSchema {
      <<function>>
    }
    class github_com_zopdev_reqflow_equalFuzzy {
      <<function>>
    }
  }
  namespace github_com_zopdev_reqflow_render {
    class github_com_zopdev_reqflow_render_APIMapRenderer {
      +Render()
    }
    class github_com_zopdev_reqflow_render_endpoint {
      <<handler>>
      +Method string
      +Path string
      +Handler string
      +Request string
      +Response string
      +File string
      +Line int
    }
    class github_com_zopdev_reqflow_render_C4Renderer {
      +Render()
    }
    class github_com_zopdev_reqflow_render_safePUMLID {
      <<function>>
    }
    class github_com_zopdev_reqflow_render_DataFlowRenderer {
      +Render()
    }
    class github_com_zopdev_reqflow_render_DOTRenderer {
      +Render()
      +renderNode()
    }
    class github_com_zopdev_reqflow_render_DSMRenderer {
      +Render()
    }
    class github_com_zopdev_reqflow_render_EmbedRenderer {
      +Render()
    }
    class github_com_zopdev_reqflow_render_ExcalidrawRenderer {
      +Render()
    }
    class github_com_zopdev_reqflow_render_excalidrawFile {
      <<model>>
      +Type string
      +Version int
      +Source string
      +Elements github.com/thzgajendra/reqflow/render.excalidrawElement
      +AppState map[string]any
    }
    class github_com_zopdev_reqflow_render_excalidrawElement {
      <<model>>
      +ID string
      +Type string
      +X float64
      +Y float64
      +Width float64
      +Height float64
      +StrokeColor string
      +BackgroundColor string
      +FillStyle string
      +StrokeWidth int
      +Roughness int
      +Opacity int
      +Text string
      +FontSize int
      +FontFamily int
      +TextAlign string
      +VerticalAlign string
      +ContainerID string
      +BoundElements github.com/thzgajendra/reqflow/render.bound
      +Points float64
      +StartBinding github.com/thzgajendra/reqflow/render.binding
      +EndBinding github.com/thzgajendra/reqflow/render.binding
      +StartArrowhead string
      +EndArrowhead string
    }
    class github_com_zopdev_reqflow_render_bound {
      <<model>>
      +ID string
      +Type string
    }
    class github_com_zopdev_reqflow_render_binding {
      <<model>>
      +ElementID string
      +Focus float64
      +Gap int
    }
    class github_com_zopdev_reqflow_render_nodeEntry {
      +id string
      +node github.com/thzgajendra/reqflow.Node
    }
    class github_com_zopdev_reqflow_render_genID {
      <<function>>
    }
    class github_com_zopdev_reqflow_render_HTMLRenderer {
      +Render()
    }
    class github_com_zopdev_reqflow_render_InteractiveRenderer {
      +Render()
    }
    class github_com_zopdev_reqflow_render_cyNode {
      <<model>>
      +ID string
      +Label string
      +Kind string
      +Pkg string
      +File string
      +Line int
      +Methods string
      +Fields string
      +Meta map[string]string
      +Parent string
    }
    class github_com_zopdev_reqflow_render_cyEdge {
      <<model>>
      +Source string
      +Target string
      +Kind string
    }
    class github_com_zopdev_reqflow_render_cyGraph {
      <<model>>
      +Nodes github.com/thzgajendra/reqflow/render.cyNode
      +Edges github.com/thzgajendra/reqflow/render.cyEdge
      +Clusters string
    }
    class github_com_zopdev_reqflow_render_JSONRenderer {
      +Render()
    }
    class github_com_zopdev_reqflow_render_MarkdownRenderer {
      +Render()
    }
    class github_com_zopdev_reqflow_render_kindIcon {
      <<function>>
    }
    class github_com_zopdev_reqflow_render_kindPriority {
      <<function>>
    }
    class github_com_zopdev_reqflow_render_Renderer {
      <<interface>>
      +Render()
    }
    class github_com_zopdev_reqflow_render_MermaidRenderer {
      +Render()
      +renderNode()
    }
    class github_com_zopdev_reqflow_render_sanitizeID {
      <<function>>
    }
    class github_com_zopdev_reqflow_render_sanitizeTypeName {
      <<function>>
    }
    class github_com_zopdev_reqflow_render_PDFRenderer {
      +Render()
    }
    class github_com_zopdev_reqflow_render_ThreeRenderer {
      +Render()
    }
    class github_com_zopdev_reqflow_render_threeNode {
      <<model>>
      +ID string
      +Name string
      +Kind string
      +Pkg string
      +Val int
    }
    class github_com_zopdev_reqflow_render_threeEdge {
      <<model>>
      +Source string
      +Target string
      +Kind string
    }
    class github_com_zopdev_reqflow_render_threeGraph {
      <<model>>
      +Nodes github.com/thzgajendra/reqflow/render.threeNode
      +Links github.com/thzgajendra/reqflow/render.threeEdge
    }
    class github_com_zopdev_reqflow_render_TimelineRenderer {
      +Snapshots github.com/thzgajendra/reqflow.EvolutionSnapshot
      +Render()
    }
  }
  namespace github_com_zopdev_reqflow_cmd {
    class github_com_zopdev_reqflow_cmd_runAnalysis {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_analysisFlags {
      +Deadcode bool
      +Cycles bool
      +Metrics bool
      +ErrCheck bool
      +Security bool
      +TechDebt bool
      +CoverFile string
      +Constructors bool
      +Diff string
      +AI bool
    }
    class github_com_zopdev_reqflow_cmd_runDeadcodeCheck {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_runCycleCheck {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_runMetricsCheck {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_runErrCheck {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_runSecurityCheck {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_runTechDebtCheck {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_runCoverageCheck {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_runConstructorCheck {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_runDiffCheck {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_runAIReview {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_import_http_post {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_loadPackages {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_enforceThresholds {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_generateInitConfig {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_Execute {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_handleStitch {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_runVetRules {
      <<function>>
    }
    class github_com_zopdev_reqflow_cmd_startServer {
      <<function>>
    }
  }
  namespace github_com_zopdev_reqflow_cmd_reqflow {
    class github_com_zopdev_reqflow_cmd_reqflow_main {
      <<function>>
    }
  }
  github_com_zopdev_reqflow_render_APIMapRenderer ..|> github_com_zopdev_reqflow_render_Renderer : implements
  github_com_zopdev_reqflow_render_ThreeRenderer ..|> github_com_zopdev_reqflow_render_Renderer : implements
  github_com_zopdev_reqflow_render_ExcalidrawRenderer ..|> github_com_zopdev_reqflow_render_Renderer : implements
  github_com_zopdev_reqflow_render_MermaidRenderer ..|> github_com_zopdev_reqflow_render_Renderer : implements
  github_com_zopdev_reqflow_render_TimelineRenderer ..|> github_com_zopdev_reqflow_render_Renderer : implements
  github_com_zopdev_reqflow_render_C4Renderer ..|> github_com_zopdev_reqflow_render_Renderer : implements
  github_com_zopdev_reqflow_render_EmbedRenderer ..|> github_com_zopdev_reqflow_render_Renderer : implements
  github_com_zopdev_reqflow_render_HTMLRenderer ..|> github_com_zopdev_reqflow_render_Renderer : implements
  github_com_zopdev_reqflow_render_InteractiveRenderer ..|> github_com_zopdev_reqflow_render_Renderer : implements
  github_com_zopdev_reqflow_render_JSONRenderer ..|> github_com_zopdev_reqflow_render_Renderer : implements
  github_com_zopdev_reqflow_render_DOTRenderer ..|> github_com_zopdev_reqflow_render_Renderer : implements
  github_com_zopdev_reqflow_render_DSMRenderer ..|> github_com_zopdev_reqflow_render_Renderer : implements
  github_com_zopdev_reqflow_render_DataFlowRenderer ..|> github_com_zopdev_reqflow_render_Renderer : implements
  github_com_zopdev_reqflow_render_MarkdownRenderer ..|> github_com_zopdev_reqflow_render_Renderer : implements
  github_com_zopdev_reqflow_render_PDFRenderer ..|> github_com_zopdev_reqflow_render_Renderer : implements

  %% Color Coding Layers
  classDef handler fill:#d4edda,stroke:#28a745,color:#155724
  classDef service fill:#cce5ff,stroke:#007bff,color:#004085
  classDef store fill:#ffeeba,stroke:#ffc107,color:#856404
  classDef model fill:#f8d7da,stroke:#dc3545,color:#721c24
  classDef event fill:#e2e3e5,stroke:#343a40,stroke-dasharray: 5 5,color:#343a40
  classDef middleware fill:#fff3cd,stroke:#856404,stroke-dasharray: 3 3,color:#856404
  classDef grpc fill:#d1ecf1,stroke:#0c5460,color:#0c5460
  classDef infra fill:#e8daef,stroke:#6c3483,color:#6c3483
  classDef diffnew fill:#d4edda,stroke:#28a745,color:#155724,stroke-width:4px,stroke-dasharray: 5 5
  classDef diffdel fill:#f8d7da,stroke:#dc3545,color:#721c24,stroke-width:4px,stroke-dasharray: 5 5
  classDef coverCritical fill:#f8d7da,stroke:#dc3545,color:#721c24,stroke-width:3px
  classDef coverLow fill:#fff3cd,stroke:#ffc107,color:#856404,stroke-width:3px
  classDef coverHealthy fill:#d4edda,stroke:#28a745,color:#155724,stroke-width:3px
  classDef churnHot fill:#f8d7da,stroke:#dc3545,color:#721c24,stroke-width:4px
  classDef churnWarm fill:#fff3cd,stroke:#ffc107,color:#856404,stroke-width:3px
  classDef churnCold fill:#cce5ff,stroke:#007bff,color:#004085,stroke-width:2px
  classDef impactDirect fill:#f8d7da,stroke:#dc3545,color:#721c24,stroke-width:4px,stroke-dasharray: 8 4
  classDef impactIndirect fill:#fff3cd,stroke:#ffc107,color:#856404,stroke-width:3px,stroke-dasharray: 4 4
  click github_com_zopdev_reqflow_ParseChurnTotal href "vscode://file/Users/zopdev/reqflow/gitchurn.go:108" "Open in VSCode"
  click github_com_zopdev_reqflow_Graph href "vscode://file/Users/zopdev/reqflow/graph.go:75" "Open in VSCode"
  click github_com_zopdev_reqflow_StitchWithServiceMap href "vscode://file/Users/zopdev/reqflow/stitch.go:71" "Open in VSCode"
  click github_com_zopdev_reqflow_extractTagValue href "vscode://file/Users/zopdev/reqflow/tablemap.go:133" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_runAnalysis href "vscode://file/Users/zopdev/reqflow/cmd/analyze.go:14" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_runMetricsCheck href "vscode://file/Users/zopdev/reqflow/cmd/analyze.go:104" "Open in VSCode"
  click github_com_zopdev_reqflow_DetectCycles href "vscode://file/Users/zopdev/reqflow/cycles.go:10" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractChurn href "vscode://file/Users/zopdev/reqflow/gitchurn.go:12" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractInfraTopo href "vscode://file/Users/zopdev/reqflow/infratopo.go:15" "Open in VSCode"
  click github_com_zopdev_reqflow_shouldIgnorePackage href "vscode://file/Users/zopdev/reqflow/parser.go:173" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_handleStitch href "vscode://file/Users/zopdev/reqflow/cmd/root.go:244" "Open in VSCode"
  click github_com_zopdev_reqflow_GetChurnSummary href "vscode://file/Users/zopdev/reqflow/gitchurn.go:85" "Open in VSCode"
  click github_com_zopdev_reqflow_SecurityIssue href "vscode://file/Users/zopdev/reqflow/deep_analysis.go:273" "Open in VSCode"
  click github_com_zopdev_reqflow_DetectSecurityIssues href "vscode://file/Users/zopdev/reqflow/deep_analysis.go:283" "Open in VSCode"
  click github_com_zopdev_reqflow_SwallowedError href "vscode://file/Users/zopdev/reqflow/errcheck.go:16" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractMiddleware href "vscode://file/Users/zopdev/reqflow/middleware.go:16" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractProto href "vscode://file/Users/zopdev/reqflow/protoparse.go:21" "Open in VSCode"
  click github_com_zopdev_reqflow_parseProtoFile href "vscode://file/Users/zopdev/reqflow/protoparse.go:41" "Open in VSCode"
  click github_com_zopdev_reqflow_render_DOTRenderer href "vscode://file/Users/zopdev/reqflow/render/dot.go:11" "Open in VSCode"
  click github_com_zopdev_reqflow_render_ThreeRenderer href "vscode://file/Users/zopdev/reqflow/render/three.go:13" "Open in VSCode"
  click github_com_zopdev_reqflow_extractTypeName href "vscode://file/Users/zopdev/reqflow/apimap.go:160" "Open in VSCode"
  click github_com_zopdev_reqflow_dockerComposeFile href "vscode://file/Users/zopdev/reqflow/infratopo.go:100" "Open in VSCode"
  click github_com_zopdev_reqflow_getModulePath href "vscode://file/Users/zopdev/reqflow/parser.go:428" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractPRImpact href "vscode://file/Users/zopdev/reqflow/primpact.go:11" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_import_http_post href "vscode://file/Users/zopdev/reqflow/cmd/analyze.go:258" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_generateInitConfig href "vscode://file/Users/zopdev/reqflow/cmd/init.go:50" "Open in VSCode"
  click github_com_zopdev_reqflow_applyFocus href "vscode://file/Users/zopdev/reqflow/focus.go:7" "Open in VSCode"
  click github_com_zopdev_reqflow_parseK8sManifest href "vscode://file/Users/zopdev/reqflow/infratopo.go:224" "Open in VSCode"
  click github_com_zopdev_reqflow_sanitizePath href "vscode://file/Users/zopdev/reqflow/infratopo.go:315" "Open in VSCode"
  class github_com_zopdev_reqflow_attributeValue model
  click github_com_zopdev_reqflow_attributeValue href "vscode://file/Users/zopdev/reqflow/otel.go:47" "Open in VSCode"
  click github_com_zopdev_reqflow_handleTypeSpec href "vscode://file/Users/zopdev/reqflow/parser.go:183" "Open in VSCode"
  click github_com_zopdev_reqflow_PrintSummary href "vscode://file/Users/zopdev/reqflow/summary.go:13" "Open in VSCode"
  click github_com_zopdev_reqflow_render_EmbedRenderer href "vscode://file/Users/zopdev/reqflow/render/embed.go:14" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_runDeadcodeCheck href "vscode://file/Users/zopdev/reqflow/cmd/analyze.go:60" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_runErrCheck href "vscode://file/Users/zopdev/reqflow/cmd/analyze.go:121" "Open in VSCode"
  click github_com_zopdev_reqflow_toSnakeCase href "vscode://file/Users/zopdev/reqflow/tablemap.go:168" "Open in VSCode"
  click github_com_zopdev_reqflow_DetectConcurrency href "vscode://file/Users/zopdev/reqflow/deep_analysis.go:22" "Open in VSCode"
  click github_com_zopdev_reqflow_extractDefault href "vscode://file/Users/zopdev/reqflow/envmap.go:111" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractEvolution href "vscode://file/Users/zopdev/reqflow/evolution.go:22" "Open in VSCode"
  click github_com_zopdev_reqflow_gitAuthors href "vscode://file/Users/zopdev/reqflow/gitblame.go:68" "Open in VSCode"
  click github_com_zopdev_reqflow_spanDurationMs href "vscode://file/Users/zopdev/reqflow/otel.go:152" "Open in VSCode"
  click github_com_zopdev_reqflow_render_C4Renderer href "vscode://file/Users/zopdev/reqflow/render/c4.go:12" "Open in VSCode"
  class github_com_zopdev_reqflow_render_binding model
  click github_com_zopdev_reqflow_render_binding href "vscode://file/Users/zopdev/reqflow/render/excalidraw.go:58" "Open in VSCode"
  click github_com_zopdev_reqflow_resolveNodeID href "vscode://file/Users/zopdev/reqflow/callgraph.go:94" "Open in VSCode"
  click github_com_zopdev_reqflow_extractEvents href "vscode://file/Users/zopdev/reqflow/events.go:29" "Open in VSCode"
  click github_com_zopdev_reqflow_render_ExcalidrawRenderer href "vscode://file/Users/zopdev/reqflow/render/excalidraw.go:16" "Open in VSCode"
  class github_com_zopdev_reqflow_render_bound model
  click github_com_zopdev_reqflow_render_bound href "vscode://file/Users/zopdev/reqflow/render/excalidraw.go:53" "Open in VSCode"
  click github_com_zopdev_reqflow_render_genID href "vscode://file/Users/zopdev/reqflow/render/excalidraw.go:214" "Open in VSCode"
  class github_com_zopdev_reqflow_render_cyEdge model
  click github_com_zopdev_reqflow_render_cyEdge href "vscode://file/Users/zopdev/reqflow/render/interactive.go:29" "Open in VSCode"
  click github_com_zopdev_reqflow_render_MarkdownRenderer href "vscode://file/Users/zopdev/reqflow/render/markdown.go:12" "Open in VSCode"
  click github_com_zopdev_reqflow_render_sanitizeTypeName href "vscode://file/Users/zopdev/reqflow/render/mermaid.go:216" "Open in VSCode"
  click github_com_zopdev_reqflow_extractServiceName href "vscode://file/Users/zopdev/reqflow/otel.go:143" "Open in VSCode"
  click github_com_zopdev_reqflow_GetSummaryHTML href "vscode://file/Users/zopdev/reqflow/summary.go:93" "Open in VSCode"
  click github_com_zopdev_reqflow_render_TimelineRenderer href "vscode://file/Users/zopdev/reqflow/render/timeline.go:13" "Open in VSCode"
  click github_com_zopdev_reqflow_LoadCoverageProfile href "vscode://file/Users/zopdev/reqflow/deep_analysis.go:91" "Open in VSCode"
  click github_com_zopdev_reqflow_findClosestNode href "vscode://file/Users/zopdev/reqflow/deep_analysis.go:210" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractContributors href "vscode://file/Users/zopdev/reqflow/gitblame.go:12" "Open in VSCode"
  class github_com_zopdev_reqflow_resource handler
  click github_com_zopdev_reqflow_resource href "vscode://file/Users/zopdev/reqflow/otel.go:22" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_runSecurityCheck href "vscode://file/Users/zopdev/reqflow/cmd/analyze.go:138" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_enforceThresholds href "vscode://file/Users/zopdev/reqflow/cmd/analyze.go:279" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_runVetRules href "vscode://file/Users/zopdev/reqflow/cmd/root.go:296" "Open in VSCode"
  click github_com_zopdev_reqflow_DetectMissingConstructors href "vscode://file/Users/zopdev/reqflow/deep_analysis.go:237" "Open in VSCode"
  class github_com_zopdev_reqflow_render_threeEdge model
  click github_com_zopdev_reqflow_render_threeEdge href "vscode://file/Users/zopdev/reqflow/render/three.go:23" "Open in VSCode"
  class github_com_zopdev_reqflow_render_threeGraph model
  click github_com_zopdev_reqflow_render_threeGraph href "vscode://file/Users/zopdev/reqflow/render/three.go:29" "Open in VSCode"
  click github_com_zopdev_reqflow_NewGraph href "vscode://file/Users/zopdev/reqflow/graph.go:82" "Open in VSCode"
  click github_com_zopdev_reqflow_ParseOptions href "vscode://file/Users/zopdev/reqflow/parser.go:14" "Open in VSCode"
  click github_com_zopdev_reqflow_crossReferenceGRPC href "vscode://file/Users/zopdev/reqflow/protoparse.go:170" "Open in VSCode"
  click github_com_zopdev_reqflow_render_PDFRenderer href "vscode://file/Users/zopdev/reqflow/render/pdf.go:15" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractAPIMap href "vscode://file/Users/zopdev/reqflow/apimap.go:15" "Open in VSCode"
  click github_com_zopdev_reqflow_DetectSwallowedErrors href "vscode://file/Users/zopdev/reqflow/errcheck.go:25" "Open in VSCode"
  click github_com_zopdev_reqflow_getNodeIDSet href "vscode://file/Users/zopdev/reqflow/evolution.go:134" "Open in VSCode"
  click github_com_zopdev_reqflow_NodeMetrics href "vscode://file/Users/zopdev/reqflow/metrics.go:9" "Open in VSCode"
  click github_com_zopdev_reqflow_resolveDependencies href "vscode://file/Users/zopdev/reqflow/parser.go:414" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractTableMap href "vscode://file/Users/zopdev/reqflow/tablemap.go:15" "Open in VSCode"
  click github_com_zopdev_reqflow_render_DSMRenderer href "vscode://file/Users/zopdev/reqflow/render/dsm.go:13" "Open in VSCode"
  class github_com_zopdev_reqflow_render_cyGraph model
  click github_com_zopdev_reqflow_render_cyGraph href "vscode://file/Users/zopdev/reqflow/render/interactive.go:35" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractCallGraph href "vscode://file/Users/zopdev/reqflow/callgraph.go:17" "Open in VSCode"
  click github_com_zopdev_reqflow_isErrorTuple href "vscode://file/Users/zopdev/reqflow/errcheck.go:71" "Open in VSCode"
  click github_com_zopdev_reqflow_Field href "vscode://file/Users/zopdev/reqflow/graph.go:52" "Open in VSCode"
  class github_com_zopdev_reqflow_spanStatus model
  click github_com_zopdev_reqflow_spanStatus href "vscode://file/Users/zopdev/reqflow/otel.go:52" "Open in VSCode"
  class github_com_zopdev_reqflow_VSchema model
  click github_com_zopdev_reqflow_VSchema href "vscode://file/Users/zopdev/reqflow/vitess.go:11" "Open in VSCode"
  click github_com_zopdev_reqflow_render_DataFlowRenderer href "vscode://file/Users/zopdev/reqflow/render/dataflow.go:12" "Open in VSCode"
  class github_com_zopdev_reqflow_render_excalidrawElement model
  click github_com_zopdev_reqflow_render_excalidrawElement href "vscode://file/Users/zopdev/reqflow/render/excalidraw.go:26" "Open in VSCode"
  click github_com_zopdev_reqflow_render_MermaidRenderer href "vscode://file/Users/zopdev/reqflow/render/mermaid.go:16" "Open in VSCode"
  click github_com_zopdev_reqflow_DataFlow href "vscode://file/Users/zopdev/reqflow/dataflow.go:4" "Open in VSCode"
  click github_com_zopdev_reqflow_commonPrefix href "vscode://file/Users/zopdev/reqflow/depstree.go:164" "Open in VSCode"
  class github_com_zopdev_reqflow_attribute model
  click github_com_zopdev_reqflow_attribute href "vscode://file/Users/zopdev/reqflow/otel.go:42" "Open in VSCode"
  click github_com_zopdev_reqflow_detectCrossServiceEdges href "vscode://file/Users/zopdev/reqflow/stitch.go:81" "Open in VSCode"
  class github_com_zopdev_reqflow_render_endpoint handler
  click github_com_zopdev_reqflow_render_endpoint href "vscode://file/Users/zopdev/reqflow/render/apimap.go:17" "Open in VSCode"
  click github_com_zopdev_reqflow_render_kindIcon href "vscode://file/Users/zopdev/reqflow/render/markdown.go:148" "Open in VSCode"
  click github_com_zopdev_reqflow_render_sanitizeID href "vscode://file/Users/zopdev/reqflow/render/mermaid.go:206" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_analysisFlags href "vscode://file/Users/zopdev/reqflow/cmd/analyze.go:47" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractGRPC href "vscode://file/Users/zopdev/reqflow/middleware.go:78" "Open in VSCode"
  click github_com_zopdev_reqflow_matchLayer href "vscode://file/Users/zopdev/reqflow/parser.go:454" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_loadPackages href "vscode://file/Users/zopdev/reqflow/cmd/analyze.go:265" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_reqflow_main href "vscode://file/Users/zopdev/reqflow/cmd/reqflow/main.go:5" "Open in VSCode"
  click github_com_zopdev_reqflow_DetectTechDebt href "vscode://file/Users/zopdev/reqflow/deep_analysis.go:170" "Open in VSCode"
  class github_com_zopdev_reqflow_resourceSpan model
  click github_com_zopdev_reqflow_resourceSpan href "vscode://file/Users/zopdev/reqflow/otel.go:17" "Open in VSCode"
  click github_com_zopdev_reqflow_equalFuzzy href "vscode://file/Users/zopdev/reqflow/vitess.go:69" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_runConstructorCheck href "vscode://file/Users/zopdev/reqflow/cmd/analyze.go:198" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractDataFlows href "vscode://file/Users/zopdev/reqflow/dataflow.go:13" "Open in VSCode"
  click github_com_zopdev_reqflow_findEnclosingNode href "vscode://file/Users/zopdev/reqflow/envmap.go:120" "Open in VSCode"
  click github_com_zopdev_reqflow_resolveProtoMsgID href "vscode://file/Users/zopdev/reqflow/protoparse.go:161" "Open in VSCode"
  click github_com_zopdev_reqflow_render_kindPriority href "vscode://file/Users/zopdev/reqflow/render/markdown.go:173" "Open in VSCode"
  class github_com_zopdev_reqflow_VTable model
  click github_com_zopdev_reqflow_VTable href "vscode://file/Users/zopdev/reqflow/vitess.go:16" "Open in VSCode"
  click github_com_zopdev_reqflow_tagConcurrency href "vscode://file/Users/zopdev/reqflow/deep_analysis.go:69" "Open in VSCode"
  click github_com_zopdev_reqflow_sanitizeRef href "vscode://file/Users/zopdev/reqflow/evolution.go:142" "Open in VSCode"
  click github_com_zopdev_reqflow_gitCommitCount href "vscode://file/Users/zopdev/reqflow/gitchurn.go:58" "Open in VSCode"
  click github_com_zopdev_reqflow_Node href "vscode://file/Users/zopdev/reqflow/graph.go:40" "Open in VSCode"
  click github_com_zopdev_reqflow_ComputeMetrics href "vscode://file/Users/zopdev/reqflow/metrics.go:20" "Open in VSCode"
  class github_com_zopdev_reqflow_OTLPExport model
  click github_com_zopdev_reqflow_OTLPExport href "vscode://file/Users/zopdev/reqflow/otel.go:13" "Open in VSCode"
  click github_com_zopdev_reqflow_percentile href "vscode://file/Users/zopdev/reqflow/otel.go:210" "Open in VSCode"
  click github_com_zopdev_reqflow_appendUnique href "vscode://file/Users/zopdev/reqflow/apimap.go:186" "Open in VSCode"
  class github_com_zopdev_reqflow_goModule model
  click github_com_zopdev_reqflow_goModule href "vscode://file/Users/zopdev/reqflow/depstree.go:11" "Open in VSCode"
  click github_com_zopdev_reqflow_EvolutionSnapshot href "vscode://file/Users/zopdev/reqflow/evolution.go:10" "Open in VSCode"
  class github_com_zopdev_reqflow_scopeSpan model
  click github_com_zopdev_reqflow_scopeSpan href "vscode://file/Users/zopdev/reqflow/otel.go:26" "Open in VSCode"
  click github_com_zopdev_reqflow_resolveInterfaces href "vscode://file/Users/zopdev/reqflow/parser.go:378" "Open in VSCode"
  click github_com_zopdev_reqflow_extractRoutes href "vscode://file/Users/zopdev/reqflow/routes.go:14" "Open in VSCode"
  click github_com_zopdev_reqflow_render_nodeEntry href "vscode://file/Users/zopdev/reqflow/render/excalidraw.go:87" "Open in VSCode"
  click github_com_zopdev_reqflow_render_HTMLRenderer href "vscode://file/Users/zopdev/reqflow/render/html.go:11" "Open in VSCode"
  click github_com_zopdev_reqflow_LoadConfig href "vscode://file/Users/zopdev/reqflow/config.go:36" "Open in VSCode"
  click github_com_zopdev_reqflow_MissingConstructor href "vscode://file/Users/zopdev/reqflow/deep_analysis.go:229" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractEnvMap href "vscode://file/Users/zopdev/reqflow/envmap.go:15" "Open in VSCode"
  class github_com_zopdev_reqflow_span model
  click github_com_zopdev_reqflow_span href "vscode://file/Users/zopdev/reqflow/otel.go:30" "Open in VSCode"
  click github_com_zopdev_reqflow_findAndTagHandler href "vscode://file/Users/zopdev/reqflow/routes.go:49" "Open in VSCode"
  click github_com_zopdev_reqflow_render_APIMapRenderer href "vscode://file/Users/zopdev/reqflow/render/apimap.go:14" "Open in VSCode"
  click github_com_zopdev_reqflow_render_JSONRenderer href "vscode://file/Users/zopdev/reqflow/render/json.go:10" "Open in VSCode"
  click github_com_zopdev_reqflow_render_Renderer href "vscode://file/Users/zopdev/reqflow/render/mermaid.go:12" "Open in VSCode"
  click github_com_zopdev_reqflow_isBindMethod href "vscode://file/Users/zopdev/reqflow/apimap.go:133" "Open in VSCode"
  click github_com_zopdev_reqflow_extractEnvCall href "vscode://file/Users/zopdev/reqflow/envmap.go:74" "Open in VSCode"
  click github_com_zopdev_reqflow_parseDockerCompose href "vscode://file/Users/zopdev/reqflow/infratopo.go:105" "Open in VSCode"
  click github_com_zopdev_reqflow_spanMetrics href "vscode://file/Users/zopdev/reqflow/otel.go:58" "Open in VSCode"
  click github_com_zopdev_reqflow_Parse href "vscode://file/Users/zopdev/reqflow/parser.go:38" "Open in VSCode"
  click github_com_zopdev_reqflow_isHTTPHandler href "vscode://file/Users/zopdev/reqflow/parser.go:334" "Open in VSCode"
  click github_com_zopdev_reqflow_parseVitessSchema href "vscode://file/Users/zopdev/reqflow/vitess.go:24" "Open in VSCode"
  click github_com_zopdev_reqflow_render_safePUMLID href "vscode://file/Users/zopdev/reqflow/render/c4.go:90" "Open in VSCode"
  click github_com_zopdev_reqflow_isResponseMethod href "vscode://file/Users/zopdev/reqflow/apimap.go:147" "Open in VSCode"
  click github_com_zopdev_reqflow_Edge href "vscode://file/Users/zopdev/reqflow/graph.go:70" "Open in VSCode"
  click github_com_zopdev_reqflow_extractDependsOn href "vscode://file/Users/zopdev/reqflow/infratopo.go:160" "Open in VSCode"
  click github_com_zopdev_reqflow_Stitch href "vscode://file/Users/zopdev/reqflow/stitch.go:10" "Open in VSCode"
  class github_com_zopdev_reqflow_render_excalidrawFile model
  click github_com_zopdev_reqflow_render_excalidrawFile href "vscode://file/Users/zopdev/reqflow/render/excalidraw.go:18" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_runCoverageCheck href "vscode://file/Users/zopdev/reqflow/cmd/analyze.go:178" "Open in VSCode"
  click github_com_zopdev_reqflow_extractKafkaConfigTopic href "vscode://file/Users/zopdev/reqflow/events.go:135" "Open in VSCode"
  click github_com_zopdev_reqflow_matchSpanToNode href "vscode://file/Users/zopdev/reqflow/otel.go:164" "Open in VSCode"
  click github_com_zopdev_reqflow_handleFuncDecl href "vscode://file/Users/zopdev/reqflow/parser.go:293" "Open in VSCode"
  click github_com_zopdev_reqflow_extractDependencies href "vscode://file/Users/zopdev/reqflow/parser.go:364" "Open in VSCode"
  class github_com_zopdev_reqflow_render_threeNode model
  click github_com_zopdev_reqflow_render_threeNode href "vscode://file/Users/zopdev/reqflow/render/three.go:15" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_runTechDebtCheck href "vscode://file/Users/zopdev/reqflow/cmd/analyze.go:161" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_startServer href "vscode://file/Users/zopdev/reqflow/cmd/serve.go:13" "Open in VSCode"
  click github_com_zopdev_reqflow_extractTopicName href "vscode://file/Users/zopdev/reqflow/events.go:125" "Open in VSCode"
  click github_com_zopdev_reqflow_isK8sManifest href "vscode://file/Users/zopdev/reqflow/infratopo.go:214" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractOtelTrace href "vscode://file/Users/zopdev/reqflow/otel.go:68" "Open in VSCode"
  click github_com_zopdev_reqflow_average href "vscode://file/Users/zopdev/reqflow/otel.go:199" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_runAIReview href "vscode://file/Users/zopdev/reqflow/cmd/analyze.go:241" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_Execute href "vscode://file/Users/zopdev/reqflow/cmd/root.go:15" "Open in VSCode"
  class github_com_zopdev_reqflow_dockerComposeService service
  click github_com_zopdev_reqflow_dockerComposeService href "vscode://file/Users/zopdev/reqflow/infratopo.go:89" "Open in VSCode"
  click github_com_zopdev_reqflow_render_InteractiveRenderer href "vscode://file/Users/zopdev/reqflow/render/interactive.go:14" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_runDiffCheck href "vscode://file/Users/zopdev/reqflow/cmd/analyze.go:215" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractDepTree href "vscode://file/Users/zopdev/reqflow/depstree.go:23" "Open in VSCode"
  click github_com_zopdev_reqflow_ExtractGoModDeps href "vscode://file/Users/zopdev/reqflow/errcheck.go:85" "Open in VSCode"
  click github_com_zopdev_reqflow_buildSnapshot href "vscode://file/Users/zopdev/reqflow/evolution.go:74" "Open in VSCode"
  click github_com_zopdev_reqflow_cmd_runCycleCheck href "vscode://file/Users/zopdev/reqflow/cmd/analyze.go:83" "Open in VSCode"
  click github_com_zopdev_reqflow_authorCount href "vscode://file/Users/zopdev/reqflow/gitblame.go:36" "Open in VSCode"
  click github_com_zopdev_reqflow_k8sManifest href "vscode://file/Users/zopdev/reqflow/infratopo.go:185" "Open in VSCode"
  click github_com_zopdev_reqflow_ReqflowConfig href "vscode://file/Users/zopdev/reqflow/config.go:11" "Open in VSCode"
  click github_com_zopdev_reqflow_parseDockerfile href "vscode://file/Users/zopdev/reqflow/infratopo.go:42" "Open in VSCode"
  class github_com_zopdev_reqflow_render_cyNode model
  click github_com_zopdev_reqflow_render_cyNode href "vscode://file/Users/zopdev/reqflow/render/interactive.go:16" "Open in VSCode"
  click github_com_zopdev_reqflow_TechDebt href "vscode://file/Users/zopdev/reqflow/deep_analysis.go:160" "Open in VSCode"
