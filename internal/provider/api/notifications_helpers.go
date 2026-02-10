package api

var ValidBranchesToBeNotified = []string{
	"all",
	"default",
	"protected",
	"default_and_protected",
}

var ValidLabelsToBeNotifiedBehavior = []string{
	"match_any",
	"match_all",
}
