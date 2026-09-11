import re, sys
from PIL import Image

SRC = "/home/haris/Downloads/gopher_ansi_truecolor.ans"

with open(SRC, "r", encoding="utf-8", errors="replace") as f:
    raw = f.read()

lines = raw.split("\n")
lines = [l for l in lines if l.strip("\r") != ""]

cell_re = re.compile(r"\x1b\[38;2;(\d+);(\d+);(\d+)m\x1b\[48;2;(\d+);(\d+);(\d+)m.")

grid_rows = []
widths = set()
for line in lines:
    cells = cell_re.findall(line)
    if not cells:
        continue
    row_top = []
    row_bot = []
    for fr, fg, fb, br, bg, bb in cells:
        row_top.append((int(fr), int(fg), int(fb)))
        row_bot.append((int(br), int(bg), int(bb)))
    grid_rows.append(row_top)
    grid_rows.append(row_bot)
    widths.add(len(cells))

print("distinct widths:", widths, file=sys.stderr)
W = max(widths)
H = len(grid_rows)
print("pixel grid:", W, "x", H, file=sys.stderr)

img = Image.new("RGB", (W, H))
px = img.load()
for y, row in enumerate(grid_rows):
    for x, c in enumerate(row):
        px[x, y] = c

img.save("/tmp/claude-1000/-home-haris-Projects-test/662d6c75-6f9d-4d79-be10-f49f54f01244/scratchpad/gopher_full.png")

# --- Determine background color by sampling corners ---
corners = [px[0,0], px[W-1,0], px[0,H-1], px[W-1,H-1]]
print("corners:", corners, file=sys.stderr)
bg = corners[0]

def dist(a, b):
    return sum((a[i]-b[i])**2 for i in range(3)) ** 0.5

THRESH = 40

# --- Crop to bounding box of non-background pixels ---
minx, miny, maxx, maxy = W, H, 0, 0
for y in range(H):
    for x in range(W):
        if dist(px[x,y], bg) > THRESH:
            minx = min(minx, x); maxx = max(maxx, x)
            miny = min(miny, y); maxy = max(maxy, y)

print("bbox:", minx, miny, maxx, maxy, file=sys.stderr)
cropped = img.crop((minx, miny, maxx+1, maxy+1))
cropped.save("/tmp/claude-1000/-home-haris-Projects-test/662d6c75-6f9d-4d79-be10-f49f54f01244/scratchpad/gopher_cropped.png")
print("cropped size:", cropped.size, file=sys.stderr)
