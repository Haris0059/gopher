from PIL import Image

SCRATCH = "/tmp/claude-1000/-home-haris-Projects-test/662d6c75-6f9d-4d79-be10-f49f54f01244/scratchpad"

img = Image.open(f"{SCRATCH}/gopher_cropped.png").convert("RGB")
W, H = img.size

# Target: ~30 terminal columns wide (fits welcome box left column).
# Each terminal row = 2 image pixel rows (half-block trick), so keep
# height an even number and roughly preserve aspect ratio.
TARGET_W = 26
scale = TARGET_W / W
TARGET_H = round(H * scale)
if TARGET_H % 2 == 1:
    TARGET_H += 1

small = img.resize((TARGET_W, TARGET_H), Image.LANCZOS)
small.save(f"{SCRATCH}/gopher_small.png")
print("small size:", small.size)

px = small.load()
bg = px[0, 0]

def dist(a, b):
    return sum((a[i] - b[i]) ** 2 for i in range(3)) ** 0.5

THRESH = 45

def cell(fg, bg_px):
    fg_t = dist(fg, bg) <= THRESH
    bg_t = dist(bg_px, bg) <= THRESH
    if fg_t and bg_t:
        return "  "  # fully transparent cell -> two spaces (reset, no color)
    if fg_t:
        # top transparent, bottom colored -> use lower half block instead
        r, g, b = bg_px
        return f"\x1b[38;2;{r};{g};{b}m▄\x1b[0m"
    if bg_t:
        r, g, b = fg
        return f"\x1b[38;2;{r};{g};{b}m▀\x1b[0m"
    r, g, b = fg
    r2, g2, b2 = bg_px
    return f"\x1b[38;2;{r};{g};{b}m\x1b[48;2;{r2};{g2};{b2}m▀\x1b[0m"

lines = []
for y in range(0, TARGET_H, 2):
    row = []
    for x in range(TARGET_W):
        row.append(cell(px[x, y], px[x, y + 1]))
    lines.append("".join(row))

out = "\n".join(lines)
with open(f"{SCRATCH}/gopher_small.ans", "w") as f:
    f.write(out + "\n")

print("terminal rows:", len(lines), "cols:", TARGET_W)
