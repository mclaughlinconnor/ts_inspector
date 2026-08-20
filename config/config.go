package config

var MinimalStartup = false

var Concurrency = false
var Debug = false
var DelayStart = false && !MinimalStartup
var IndexingExperiementalParallelInitialIndexing = false
var LSP = true
var LogsPath = "/home/connor/Development/ts_inspector/logs/"
var SemanticSearch = true && !MinimalStartup
var SemanticSearchEnableFaiss = false
var SemanticSearchEnableFzf = true
var SemanticSearchEnableSqlite = true
var SemanticSearchIncludeFileInterestingPoints = false
var TcbExperimentalTagBasedAttributeRendering = false
var TsGo = true && !MinimalStartup
var TsGoExperimentalThingCaching = false
var TsGoPath = "/home/connor/.local/share/nvim/mason/packages/tsgo/node_modules/@typescript/native-preview/lib/tsgo.js"
