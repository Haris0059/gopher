package prompt

// Source: tools/BriefTool/prompt.ts:12-23 (BRIEF_PROACTIVE_SECTION)
//
// sendUserMessageToolName mirrors tools.BriefToolName without importing
// pkg/tools (which would create an import cycle back into pkg/prompt via
// pkg/tools' own dependency graph); keep the two constants in sync.
const sendUserMessageToolName = "SendUserMessage"

// BriefProactiveSection is the "## Talking to the user" guidance shown when
// the SendUserMessage ("Brief") tool is enabled.
var BriefProactiveSection = "## Talking to the user\n\n" +
	sendUserMessageToolName + " is where your replies go. Text outside it is visible if the user expands the detail view, but most won't — assume unread. Anything you want them to actually see goes through " + sendUserMessageToolName + ". The failure mode: the real answer lives in plain text while " + sendUserMessageToolName + ` just says "done!" — they see "done!" and miss everything.

So: every time the user says something, the reply they actually read comes through ` + sendUserMessageToolName + `. Even for "hi". Even for "thanks".

If you can answer right away, send the answer. If you need to go look — run a command, read files, check something — ack first in one line ("On it — checking the test output"), then work, then send the result. Without the ack they're staring at a spinner.

For longer work: ack → work → result. Between those, send a checkpoint when something useful happened — a decision you made, a surprise you hit, a phase boundary. Skip the filler ("running tests...") — a checkpoint earns its place by carrying information.

Keep messages tight — the decision, the file:line, the PR number. Second person always ("your config"), never third.`

// BriefSection returns the system-prompt Section for SendUserMessage
// guidance, present only when enabled() reports the tool is active.
// Uncached since brief-only mode can toggle mid-session (/brief).
// Source: constants/prompts.ts:553 — systemPromptSection('brief', getBriefSection)
func BriefSection(enabled func() bool) Section {
	return UncachedSystemPromptSection("brief", func() *string {
		if enabled == nil || !enabled() {
			return nil
		}
		s := BriefProactiveSection
		return &s
	}, "brief-only mode can toggle mid-session via /brief")
}
