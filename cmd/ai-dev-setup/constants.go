package main

const (
	cliName          = "ai-dev-setup"
	cliVersion       = "0.3.1"
	claudePackage    = "@anthropic-ai/claude-code"
	codexPackage     = "@openai/codex"
	minNodeMajor     = 18
	beginMarker      = "# >>> ai-dev-setup managed env >>>"
	endMarker        = "# <<< ai-dev-setup managed env <<<"
	wrapperBegin     = "# >>> ai-dev-setup shell wrapper >>>"
	wrapperEnd       = "# <<< ai-dev-setup shell wrapper <<<"
	defaultSkillsRef = "main"
	defaultSkillsGit = "https://github.com/ayushag-nv/ai-skills.git"
)
