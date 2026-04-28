package main

const (
	cliName          = "wolfpack"
	cliVersion       = "0.4.0"
	claudePackage    = "@anthropic-ai/claude-code"
	codexPackage     = "@openai/codex"
	minNodeMajor     = 18
	beginMarker      = "# >>> wolfpack managed env >>>"
	endMarker        = "# <<< wolfpack managed env <<<"
	wrapperBegin     = "# >>> wolfpack shell wrapper >>>"
	wrapperEnd       = "# <<< wolfpack shell wrapper <<<"
	defaultSkillsRef = "main"
	defaultSkillsGit = "https://github.com/ayushag-nv/ai-skills.git"
)
