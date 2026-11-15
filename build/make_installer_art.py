"""Draws the picture on the installer's first and last pages (164x314, 24-bit BMP, as NSIS wants).

Same look as the app: charcoal with an amber TV glyph and the wordmark in Bahnschrift.
Run from the repo root:  python build/make_installer_art.py
"""
import os
from PIL import Image, ImageDraw, ImageFont

W, H = 164, 314
S = 3  # draw large, then scale down for smooth edges
img = Image.new("RGB", (W * S, H * S))
top, bottom = (46, 44, 40), (17, 17, 17)
d = ImageDraw.Draw(img)
for y in range(H * S):
    t = y / (H * S - 1)
    d.line([(0, y), (W * S, y)], fill=tuple(round(a + (b - a) * t) for a, b in zip(top, bottom)))

# A soft warm glow behind the glyph
glow = Image.new("L", (W * S, H * S), 0)
gd = ImageDraw.Draw(glow)
cx, cy = W * S // 2, int(H * S * 0.36)
for r in range(150 * S // 2, 0, -6):
    gd.ellipse([cx - r, cy - r, cx + r, cy + r], fill=int(70 * (1 - r / (150 * S / 2))))
img = Image.composite(Image.new("RGB", img.size, (245, 182, 56)), img, glow.point(lambda v: v // 3))
d = ImageDraw.Draw(img)

amber = (245, 182, 56)
# TV glyph (same shape as the app icon)
w, h = 84 * S, 56 * S
x0, y0 = cx - w // 2, cy - h // 2 - 6 * S
d.rounded_rectangle([x0, y0, x0 + w, y0 + h], radius=11 * S, outline=amber, width=7 * S)
d.rounded_rectangle([cx - 16 * S, y0 + h + 9 * S, cx + 16 * S, y0 + h + 16 * S], radius=4 * S, fill=amber)
p = 15 * S
d.polygon([(cx - p * 0.55, cy - 6 * S - p * 0.75), (cx - p * 0.55, cy - 6 * S + p * 0.75), (cx + p * 0.75, cy - 6 * S)], fill=amber)

font_path = os.path.join(os.environ.get("WINDIR", r"C:\Windows"), "Fonts", "bahnschrift.ttf")
try:
    font = ImageFont.truetype(font_path, 26 * S)
    font.set_variation_by_name("Bold Condensed")
except Exception:
    font = ImageFont.load_default()
text = "TV SHARE"
tw = d.textlength(text, font=font)
d.text((cx - tw / 2, int(H * S * 0.62)), text, font=font, fill=(243, 241, 236))

img = img.resize((W, H), Image.LANCZOS)
here = os.path.dirname(os.path.abspath(__file__))
out = os.path.join(here, "installer", "welcome.bmp")
img.save(out, "BMP")
print("wrote", out)
