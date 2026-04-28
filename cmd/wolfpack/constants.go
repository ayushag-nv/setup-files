package main

// constants.go keeps shared package names, markers, and default sources.

const (
	cliName          = "wolfpack"
	cliVersion       = "0.5.0"
	claudePackage    = "@anthropic-ai/claude-code"
	codexPackage     = "@openai/codex"
	opencodePackage  = "opencode-ai"
	minNodeMajor     = 18
	beginMarker      = "# >>> wolfpack managed env >>>"
	endMarker        = "# <<< wolfpack managed env <<<"
	wrapperBegin     = "# >>> wolfpack shell wrapper >>>"
	wrapperEnd       = "# <<< wolfpack shell wrapper <<<"
	defaultSkillsRef = "main"
	defaultSkillsGit = "https://github.com/ayushag-nv/ai-skills.git"
)
