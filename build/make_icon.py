# Draws build/appicon.png (Wails turns it into the .exe and installer icon).
# Usage: python build/make_icon.py build/appicon.png   (needs Pillow: pip install pillow)
import sys
from PIL import Image, ImageDraw

S = 1024
SS = 4  # supersample for smooth edges
W = S * SS
img = Image.new("RGBA", (W, W), (0, 0, 0, 0))

# Background: rounded square with a vertical gradient (indigo -> blue)
grad = Image.new("RGBA", (W, W))
top, bot = (88, 110, 255), (46, 72, 230)
gd = ImageDraw.Draw(grad)
for y in range(W):
    t = y / (W - 1)
    gd.line([(0, y), (W, y)], fill=tuple(int(top[i] + (bot[i] - top[i]) * t) for i in range(3)) + (255,))
mask = Image.new("L", (W, W), 0)
ImageDraw.Draw(mask).rounded_rectangle([0, 0, W - 1, W - 1], radius=int(W * 0.225), fill=255)
img.paste(grad, (0, 0), mask)

d = ImageDraw.Draw(img)
white = (255, 255, 255, 255)
u = W / 32  # same grid as the in-app logo
lw = int(2.4 * u)
# screen
d.rounded_rectangle([5 * u, 7.5 * u, 27 * u, 22 * u], radius=int(3 * u), outline=white, width=lw)
# stand
d.rounded_rectangle([11.5 * u, 25.3 * u, 20.5 * u, 25.3 * u + lw], radius=lw // 2, fill=white)
# play triangle
d.polygon([(13.6 * u, 11.2 * u), (13.6 * u, 18.3 * u), (19.6 * u, 14.75 * u)], fill=white)

img = img.resize((S, S), Image.LANCZOS)
img.save(sys.argv[1])
print("saved", sys.argv[1])
