package action

// makes sure DryRun, Help, Log, Status, StatusJSON implements Action interface
var _ Action = &DryRun{}
var _ Action = &Help{}
var _ Action = &Log{}
var _ Action = &Status{}
