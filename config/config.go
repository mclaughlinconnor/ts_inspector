package config

var MinimalStartup = false

var Concurrency = false
var Debug = false
var DelayStart = false && !MinimalStartup
var LSP = true
var SemanticSearch = true && !MinimalStartup
var SemanticSearchEnableFaiss = false
var SemanticSearchEnableFzf = true
var SemanticSearchEnableSqlite = true
var SemanticSearchIncludeFileInterestingPoints = false
var TsGo = true && !MinimalStartup
var TsGoExperimentalThingCaching = false
