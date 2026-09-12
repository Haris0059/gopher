package tools

import (
	"strconv"
	"strings"
)

// replLanguage identifies a canonical REPL interpreter. Gopher's REPL tool
// keeps one interpreter process alive per (session, language) — see
// pkg/tools/repl_session.go — unlike the TS reference's REPLTool, which is a
// JavaScript VM sandbox wrapping other tools (TASKS.md TOOL-10) rather than a
// persistent shell/interpreter. This file has no TS source to port from.
type replLanguage string

const (
	replPython replLanguage = "python"
	replNode   replLanguage = "node"
	replRuby   replLanguage = "ruby"
	replBash   replLanguage = "bash"
)

// canonicalLanguage maps user-supplied aliases to a replLanguage, or reports
// ok=false for anything unsupported.
func canonicalLanguage(lang string) (replLanguage, bool) {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "python", "python3", "py":
		return replPython, true
	case "node", "nodejs", "js", "javascript":
		return replNode, true
	case "ruby", "rb":
		return replRuby, true
	case "bash", "sh", "shell":
		return replBash, true
	default:
		return "", false
	}
}

// SupportedREPLLanguages lists the accepted `language` values, for the tool
// schema and error messages.
var SupportedREPLLanguages = []string{"python", "node", "ruby", "bash"}

// replDriverSpec describes how to launch and frame a persistent interpreter
// for one language.
type replDriverSpec struct {
	// Argv builds the command line for exec.Command given the sentinel that
	// the driver must print, on its own line, after each code block.
	Argv func(sentinel string) []string
	// FrameCode wraps a user code block into whatever the driver's stdin
	// protocol expects for one turn. sentinel is the same value passed to
	// Argv — python/node/ruby drivers already know it via argv and ignore
	// the parameter; bash has no driver script, so it embeds the sentinel
	// directly into the framed shell code.
	FrameCode func(code, sentinel string) string
}

// pythonDriver runs code via exec() against one persistent globals() dict,
// printing tracebacks to stdout (redirected) so both stdout and errors are
// visible in tool output without extra plumbing, then the sentinel + a
// status line so Go knows whether the block raised.
const pythonDriverSrc = `
import sys, traceback
__repl_ns__ = {}
while True:
    line = sys.stdin.readline()
    if not line:
        break
    try:
        n = int(line.strip())
    except ValueError:
        continue
    src = sys.stdin.read(n)
    sys.stdin.read(1)  # trailing newline after the code block
    status = "0"
    try:
        exec(compile(src, "<repl>", "exec"), __repl_ns__)
    except SystemExit:
        raise
    except Exception:
        traceback.print_exc()
        status = "1"
    sys.stdout.flush()
    sys.stderr.flush()
    print("%s\n%s" % (sys.argv[1], status))
    sys.stdout.flush()
`

// nodeDriver evaluates each code block directly (unwrapped) against one
// persistent vm.Context and prints the sentinel after each turn. Unwrapped
// execution matters: node's vm module gives every runInContext call its own
// function-level scope, but "let"/"const" declared at a script's own top
// level bind to the context's shared global lexical environment and do
// persist across separate runInContext calls — wrapping in an IIFE (to get
// top-level await) would put them in a fresh function scope instead and
// silently drop them each call. The tradeoff: no implicit top-level await —
// a block that needs to await returns a Promise, which is awaited here, but
// a "let"/"const" it declares alongside the await stays scoped to that one
// call, same as it would in a real IIFE. Uncaught exceptions are caught and
// reported without killing the process.
const nodeDriverSrc = `
const vm = require('vm');
const ctx = vm.createContext({console, require, process, __dirname, __filename, Buffer, setTimeout, setInterval, clearTimeout, clearInterval, Promise});
const sentinel = process.argv[1];
let buf = '';
let expect = null;
process.stdin.on('data', chunk => {
  buf += chunk;
  processBuffer();
});
function processBuffer() {
  for (;;) {
    if (expect === null) {
      const nl = buf.indexOf('\n');
      if (nl === -1) return;
      const header = buf.slice(0, nl);
      buf = buf.slice(nl + 1);
      const n = parseInt(header, 10);
      if (Number.isNaN(n)) continue;
      expect = n;
    }
    if (buf.length < expect + 1) return;
    const src = buf.slice(0, expect);
    buf = buf.slice(expect + 1);
    expect = null;
    runBlock(src);
  }
}
async function runBlock(src) {
  let status = '0';
  try {
    const result = vm.runInContext(src, ctx, { filename: '<repl>' });
    if (result && typeof result.then === 'function') {
      await result;
    }
  } catch (e) {
    console.error(e && e.stack ? e.stack : String(e));
    status = '1';
  }
  process.stdout.write(sentinel + '\n' + status + '\n');
}
`

// rubyDriver eval()s code against one persistent binding, mirroring the
// python/node drivers' framing protocol.
const rubyDriverSrc = `
sentinel = ARGV[0]
b = binding
loop do
  line = STDIN.gets
  break if line.nil?
  n = line.strip.to_i rescue next
  src = STDIN.read(n)
  STDIN.read(1)
  status = "0"
  begin
    b.eval(src, "(repl)")
  rescue Exception => e
    STDERR.puts e.full_message(highlight: false)
    status = "1"
  end
  STDOUT.flush
  STDERR.flush
  puts sentinel
  puts status
  STDOUT.flush
end
`

// replDrivers maps each supported language to its launch/frame spec. bash
// has no driver script — it reuses the shell's own stdin loop, framing each
// block with a plain `echo` of the sentinel since the shell already executes
// statements sequentially against one persistent environment.
var replDrivers = map[replLanguage]replDriverSpec{
	replPython: {
		Argv: func(sentinel string) []string {
			return []string{"python3", "-u", "-c", pythonDriverSrc, sentinel}
		},
		FrameCode: func(code, _ string) string { return framedBlock(code) },
	},
	replNode: {
		Argv: func(sentinel string) []string {
			return []string{"node", "-e", nodeDriverSrc, "--", sentinel}
		},
		FrameCode: func(code, _ string) string { return framedBlock(code) },
	},
	replRuby: {
		Argv: func(sentinel string) []string {
			return []string{"ruby", "-e", rubyDriverSrc, sentinel}
		},
		FrameCode: func(code, _ string) string { return framedBlock(code) },
	},
	replBash: {
		Argv: func(_ string) []string {
			return nil // handled specially: launches getUserShell() -s
		},
		// { } groups the code in the current shell (unlike ( ), which forks
		// a subshell and would lose env/cwd/variable state), then reports
		// the block's exit status after the sentinel line.
		FrameCode: func(code, sentinel string) string {
			return "{\n" + code + "\n} ; __gopher_repl_status=$?\n" +
				"echo \"" + sentinel + "\"\necho \"$__gopher_repl_status\"\n"
		},
	},
}

// framedBlock renders the "<len>\n<code>\n" wire format the python/node/ruby
// drivers read on stdin: a decimal byte length, a newline, exactly that many
// bytes of code, then one more newline the driver consumes and discards.
func framedBlock(code string) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(len(code)))
	b.WriteByte('\n')
	b.WriteString(code)
	b.WriteByte('\n')
	return b.String()
}
