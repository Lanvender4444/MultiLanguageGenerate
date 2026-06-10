package translator

type Job struct {
	SourceText     string
	SourceFile     string
	SourceLanguage string
	TargetCode     string
	TargetName     string
	OutputDir      string
}

type Result struct {
	TargetCode string
	OutputPath string
	Error      error
}
