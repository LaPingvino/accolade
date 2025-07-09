import re

# Character names (ALL CAPS, centered, followed by dialogue)
CHARACTER = re.compile(
    r"^[ \t]*(?P<name>[A-Z][A-Z0-9 .'_-]*[A-Z0-9])[ \t]*$", re.M)

# Scene headings (INT./EXT. followed by location)
SCENE_HEADING = re.compile(
    r"^(?P<heading>(?:INT|EXT|EST|int|ext|est)\.?\s+.+?)(?:\s+-\s+(?P<time>.+?))?$", re.M)

# Action lines (general text that's not dialogue, character names, or scene headings)
ACTION = re.compile(
    r"^(?P<text>(?![ \t]*[A-Z][A-Z0-9 .'_-]*[A-Z0-9][ \t]*$)(?!(?:INT|EXT|EST|int|ext|est)\.?\s+).+?)$", re.M)

# Dialogue (text that follows a character name)
DIALOGUE = re.compile(
    r"^(?P<text>(?![ \t]*[A-Z][A-Z0-9 .'_-]*[A-Z0-9][ \t]*$)(?!(?:INT|EXT|EST|int|ext|est)\.?\s+).+?)$", re.M)

# Parentheticals (text in parentheses, usually within dialogue)
PARENTHETICAL = re.compile(
    r"^[ \t]*\((?P<text>.*?)\)[ \t]*$", re.M)

# Transitions (usually ALL CAPS, right-aligned, ending with TO:)
TRANSITION = re.compile(
    r"^[ \t]*(?P<text>[A-Z ]+TO:|FADE IN:|FADE OUT\.|CUT TO:|DISSOLVE TO:|SMASH CUT TO:|MATCH CUT TO:|JUMP CUT TO:|IRIS IN:|IRIS OUT:)[ \t]*$", re.M)

# Forced character names (starting with @)
FORCED_CHARACTER = re.compile(
    r"^@(?P<name>.+?)$", re.M)

# Forced scene headings (starting with .)
FORCED_SCENE_HEADING = re.compile(
    r"^\.(?P<heading>.+?)$", re.M)

# Forced action (starting with !)
FORCED_ACTION = re.compile(
    r"^!(?P<text>.+?)$", re.M)

# Centered text (starting and ending with >)
CENTERED = re.compile(
    r"^>(?P<text>.*?)<$", re.M)

# Page breaks (===)
PAGE_BREAK = re.compile(
    r"^[ \t]*={3,}[ \t]*$", re.M)

# Emphasis (italic) - *text*
ITALIC = re.compile(
    r"(?<!\\)\*(?P<text>[^\*\n]+?)(?<!\\)\*")

# Bold - **text**
BOLD = re.compile(
    r"(?<!\\)\*\*(?P<text>[^\*\n]+?)(?<!\\)\*\*")

# Underline - _text_
UNDERLINE = re.compile(
    r"(?<!\\)_(?P<text>[^_\n]+?)(?<!\\)_")

# Notes/Comments - [[text]]
NOTE = re.compile(
    r"\[\[(?P<text>.*?)\]\]", re.S)

# Boneyard/Omit - /*text*/
BONEYARD = re.compile(
    r"\/\*(?P<text>.*?)\*\/", re.S)

# Section headings - # text
SECTION = re.compile(
    r"^#+\s*(?P<text>.+?)$", re.M)

# Synopsis - = text
SYNOPSIS = re.compile(
    r"^=\s*(?P<text>.+?)$", re.M)

# Dual dialogue indicators
DUAL_DIALOGUE = re.compile(
    r"\^$", re.M)

# Lyrics (starting with ~)
LYRICS = re.compile(
    r"^~(?P<text>.+?)$", re.M)

# Title page elements
TITLE_PAGE = re.compile(
    r"^(?P<key>Title|Credit|Author|Authors|Source|Draft date|Date|Contact|Copyright):\s*(?P<value>.+?)$", re.M | re.I)

# Character extension (O.S., V.O., etc.)
CHARACTER_EXTENSION = re.compile(
    r"^[ \t]*(?P<name>[A-Z][A-Z0-9 .'_-]*[A-Z0-9])[ \t]*\((?P<extension>[^)]+)\)[ \t]*$", re.M)

# Scene numbers - #1# or #1A#
SCENE_NUMBER = re.compile(
    r"#(?P<number>[0-9A-Za-z.-]+)#")

# More/Cont'd indicators
MORE = re.compile(
    r"^[ \t]*\(MORE\)[ \t]*$", re.M)

CONTD = re.compile(
    r"^[ \t]*\(CONT'D\)[ \t]*$", re.M)