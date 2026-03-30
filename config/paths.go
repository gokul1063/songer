package config

type Paths struct {
	WorkflowLog string
	ErrorLog    string
	DownloadDir string
}

var AppPaths = Paths{
	WorkflowLog: "logs/workflow/workflow.log",
	ErrorLog:    "logs/error/error.log",
	DownloadDir: "downloads",
}

