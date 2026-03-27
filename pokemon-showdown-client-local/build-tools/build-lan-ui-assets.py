#!/usr/bin/env python3
import os
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parent.parent
SPRITES_DIR = ROOT / "play.pokemonshowdown.com" / "sprites"
TYPES_DIR = SPRITES_DIR / "types"
CATEGORIES_DIR = SPRITES_DIR / "categories"

TYPE_COLORS = {
    "Normal": "#A8A77A",
    "Fire": "#EE8130",
    "Water": "#6390F0",
    "Electric": "#F7D02C",
    "Grass": "#7AC74C",
    "Ice": "#96D9D6",
    "Fighting": "#C22E28",
    "Poison": "#A33EA1",
    "Ground": "#E2BF65",
    "Flying": "#A98FF3",
    "Psychic": "#F95587",
    "Bug": "#A6B91A",
    "Rock": "#B6A136",
    "Ghost": "#735797",
    "Dragon": "#6F35FC",
    "Dark": "#705746",
    "Steel": "#B7B7CE",
    "Fairy": "#D685AD",
    "Stellar": "#4AA6B5",
    "???": "#5C6670",
}

CATEGORY_COLORS = {
    "Physical": "#C24A3C",
    "Special": "#4E88C7",
    "Status": "#8A8A8A",
    "undefined": "#5C6670",
}


def _font(size: int) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    for candidate in (
        "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
        "/usr/share/fonts/truetype/liberation2/LiberationSans-Bold.ttf",
    ):
        if os.path.exists(candidate):
            return ImageFont.truetype(candidate, size)
    return ImageFont.load_default()


FONT_9 = _font(9)
FONT_10 = _font(10)


def _rounded_badge(size: tuple[int, int], label: str, fill: str, font, text_fill: str = "white") -> Image.Image:
    image = Image.new("RGBA", size, (0, 0, 0, 0))
    draw = ImageDraw.Draw(image)
    draw.rounded_rectangle((0, 0, size[0] - 1, size[1] - 1), radius=4, fill=fill, outline=(255, 255, 255, 60))
    bbox = draw.textbbox((0, 0), label, font=font)
    x = (size[0] - (bbox[2] - bbox[0])) / 2
    y = (size[1] - (bbox[3] - bbox[1])) / 2 - 1
    draw.text((x, y), label, font=font, fill=text_fill)
    return image


def _save_png(image: Image.Image, path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    image.save(path, format="PNG", optimize=True)


def build_type_badges() -> None:
    for type_name, color in TYPE_COLORS.items():
        badge = _rounded_badge((32, 14), type_name[:7], color, FONT_9)
        _save_png(badge, TYPES_DIR / f"{type_name.replace('?', '%3f')}.png")

        tera_label = type_name[0] if type_name != "???" else "?"
        tera = _rounded_badge((16, 16), tera_label, color, FONT_10)
        _save_png(tera, TYPES_DIR / f"Tera{type_name}.png")


def build_category_badges() -> None:
    for category_name, color in CATEGORY_COLORS.items():
        badge = _rounded_badge((32, 14), category_name[:8], color, FONT_9)
        _save_png(badge, CATEGORIES_DIR / f"{category_name}.png")


if __name__ == "__main__":
    build_type_badges()
    build_category_badges()
    print("Built LAN UI sprite placeholders")
