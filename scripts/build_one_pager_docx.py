"""Build the ContextRail one-page product brief from Markdown."""

from __future__ import annotations

import argparse
import hashlib
import re
from pathlib import Path

from docx import Document
from docx.enum.table import WD_ALIGN_VERTICAL, WD_TABLE_ALIGNMENT
from docx.enum.text import WD_ALIGN_PARAGRAPH
from docx.oxml import OxmlElement
from docx.oxml.ns import qn
from docx.shared import Inches, Pt, RGBColor


LATIN_FONT = "Arial"
# LibreOffice's headless renderer can resolve this macOS font consistently when
# the repository QA fontconfig is supplied.
CJK_FONT = "Arial Unicode MS"
CONTENT_WIDTH = 7.3


def set_font(run, size=None, bold=None, italic=None, color="000000"):
    run.font.name = LATIN_FONT
    rpr = run._element.get_or_add_rPr()
    rpr.rFonts.set(qn("w:ascii"), LATIN_FONT)
    rpr.rFonts.set(qn("w:hAnsi"), LATIN_FONT)
    rpr.rFonts.set(qn("w:eastAsia"), CJK_FONT)
    # Preserve Traditional Chinese script selection in LibreOffice's
    # headless renderer; otherwise it may apply Arial and show empty boxes.
    rpr.rFonts.set(qn("w:hint"), "eastAsia")
    lang = rpr.find(qn("w:lang"))
    if lang is None:
        lang = OxmlElement("w:lang")
        rpr.append(lang)
    lang.set(qn("w:eastAsia"), "zh-TW")
    if size is not None:
        run.font.size = Pt(size)
    if bold is not None:
        run.bold = bold
    if italic is not None:
        run.italic = italic
    run.font.color.rgb = RGBColor.from_string(color)


def xml_child(parent, tag, attrs):
    node = OxmlElement(tag)
    for key, value in attrs.items():
        node.set(qn(key), str(value))
    parent.append(node)
    return node


def inline(paragraph, text, size=9.5, color="000000"):
    pattern = r"(\*\*[^*]+\*\*|`[^`]+`|\[[^\]]+\]\([^)]+\))"
    for part in re.split(pattern, text):
        if not part:
            continue
        strong = part.startswith("**") and part.endswith("**")
        code = part.startswith("`") and part.endswith("`")
        link = re.fullmatch(r"\[([^\]]+)\]\(([^)]+)\)", part)
        value = part[2:-2] if strong else part[1:-1] if code else link.group(1) if link else part
        run = paragraph.add_run(value)
        set_font(run, size=size, bold=strong, color="1F4E79" if link else color)


def set_cell_format(cell, text, fill, font_size=8.8, bold=False, color="000000"):
    tc_pr = cell._tc.get_or_add_tcPr()
    xml_child(tc_pr, "w:shd", {"w:fill": fill})
    cell.vertical_alignment = WD_ALIGN_VERTICAL.CENTER
    p = cell.paragraphs[0]
    p.paragraph_format.space_before = Pt(0)
    p.paragraph_format.space_after = Pt(0)
    p.paragraph_format.line_spacing = 1.02
    p.alignment = WD_ALIGN_PARAGRAPH.LEFT
    for run in list(p.runs):
        run._element.getparent().remove(run._element)
    inline(p, text, size=font_size, color=color)
    if bold:
        for run in p.runs:
            run.bold = True


def make_table(doc, rows):
    cols = len(rows[0])
    if cols == 2:
        widths = [1.55, CONTENT_WIDTH - 1.55]
    elif cols == 3:
        widths = [1.05, 2.55, CONTENT_WIDTH - 3.6]
    else:
        widths = [CONTENT_WIDTH / cols] * cols

    table = doc.add_table(rows=0, cols=cols)
    table.alignment = WD_TABLE_ALIGNMENT.CENTER
    table.autofit = False
    for column, width in zip(table.columns, widths):
        column.width = Inches(width)

    tbl_pr = table._tbl.tblPr
    borders = xml_child(tbl_pr, "w:tblBorders", {})
    for side in ("top", "left", "bottom", "right", "insideH", "insideV"):
        xml_child(borders, "w:" + side, {"w:val": "single", "w:sz": 4, "w:color": "D9D9D9"})
    margins = xml_child(tbl_pr, "w:tblCellMar", {})
    for side in ("top", "bottom", "left", "right"):
        xml_child(margins, "w:" + side, {"w:w": 55, "w:type": "dxa"})

    for row_index, values in enumerate(rows):
        row = table.add_row()
        tr_pr = row._tr.get_or_add_trPr()
        xml_child(tr_pr, "w:cantSplit", {})
        if row_index == 0:
            xml_child(tr_pr, "w:tblHeader", {})
        for col_index, (cell, value, width) in enumerate(zip(row.cells, values, widths)):
            cell.width = Inches(width)
            is_header = row_index == 0
            # Key/value tables use a pale-blue key column rather than a header row.
            if cols == 2 and row_index > 0:
                fill = "EAF1F8" if col_index == 0 else "FFFFFF"
                set_cell_format(cell, value, fill, font_size=8.25, bold=col_index == 0)
            else:
                fill = "23374D" if is_header else ("F3F5F7" if row_index % 2 == 0 else "FFFFFF")
                set_cell_format(cell, value, fill, font_size=8.3, bold=is_header, color="FFFFFF" if is_header else "000000")

def configure(doc):
    for border in doc.styles.element.xpath(".//w:pBdr"):
        border.getparent().remove(border)
    for fonts in doc.styles.element.xpath(".//w:rFonts"):
        for key in ("asciiTheme", "hAnsiTheme", "eastAsiaTheme", "cstheme"):
            fonts.attrib.pop(qn("w:" + key), None)

    section = doc.sections[0]
    section.page_width = Inches(8.5)
    section.page_height = Inches(11)
    section.left_margin = Inches(0.6)
    section.right_margin = Inches(0.6)
    section.top_margin = Inches(0.35)
    section.bottom_margin = Inches(0.35)
    section.footer_distance = Inches(0.15)

    for name in ("Normal", "Title", "Subtitle", "Heading 1", "Heading 2", "Heading 3", "List Bullet", "List Number"):
        style = doc.styles[name]
        style.font.name = LATIN_FONT
        style.font.color.rgb = RGBColor(0, 0, 0)
        rfonts = style.element.get_or_add_rPr().rFonts
        rfonts.set(qn("w:ascii"), LATIN_FONT)
        rfonts.set(qn("w:hAnsi"), LATIN_FONT)
        rfonts.set(qn("w:eastAsia"), CJK_FONT)
        rfonts.set(qn("w:hint"), "eastAsia")
        style.paragraph_format.alignment = WD_ALIGN_PARAGRAPH.LEFT
        style.paragraph_format.widow_control = True

    normal = doc.styles["Normal"]
    normal.font.size = Pt(9.2)
    normal.paragraph_format.line_spacing = 1.0
    normal.paragraph_format.space_after = Pt(2)

    title = doc.styles["Title"]
    title.font.size = Pt(19.5)
    title.font.bold = True
    title.paragraph_format.space_after = Pt(1)

    subtitle = doc.styles["Subtitle"]
    subtitle.font.size = Pt(8.6)
    subtitle.font.italic = False
    subtitle.paragraph_format.space_after = Pt(4)

    for name, size in (("Heading 1", 11.1), ("Heading 2", 9.9), ("Heading 3", 9.5)):
        style = doc.styles[name]
        style.font.size = Pt(size)
        style.font.bold = True
        style.paragraph_format.space_before = Pt(3)
        style.paragraph_format.space_after = Pt(1)
        style.paragraph_format.keep_with_next = True

    for name in ("List Bullet", "List Number"):
        style = doc.styles[name]
        style.font.size = Pt(8.7)
        style.paragraph_format.left_indent = Inches(0.18)
        style.paragraph_format.first_line_indent = Inches(-0.12)
        style.paragraph_format.space_after = Pt(0)
        style.paragraph_format.line_spacing = 1.0

    footer = section.footer.paragraphs[0]
    footer.alignment = WD_ALIGN_PARAGRAPH.RIGHT
    set_font(footer.add_run("ContextRail Product Brief v2.3   |   "), size=7.8, color="666666")
    field = OxmlElement("w:fldSimple")
    field.set(qn("w:instr"), "PAGE")
    footer._p.append(field)


def build(source: Path, output: Path):
    content = source.read_text(encoding="utf-8")
    lines = content.splitlines()
    doc = Document()
    configure(doc)
    index = 0
    while index < len(lines):
        line = lines[index].strip()
        if not line:
            index += 1
            continue
        if line.startswith("|"):
            rows = []
            while index < len(lines) and lines[index].strip().startswith("|"):
                cells = [part.strip() for part in lines[index].strip().strip("|").split("|")]
                if not all(re.fullmatch(r":?-+:?", cell) for cell in cells):
                    rows.append(cells)
                index += 1
            if not rows or any(len(row) != len(rows[0]) for row in rows):
                raise ValueError("Inconsistent table column count")
            make_table(doc, rows)
            continue
        heading = re.match(r"^(#{1,3})\s+(.+)$", line)
        if heading:
            level = len(heading.group(1))
            style = "Title" if level == 1 else "Heading " + str(level - 1)
            p = doc.add_paragraph(style=style)
            inline(p, heading.group(2), size=21 if level == 1 else 11.6, color="000000")
        elif line.startswith("- "):
            p = doc.add_paragraph(style="List Bullet")
            inline(p, line[2:], size=9.1)
        elif re.match(r"^\d+\. ", line):
            p = doc.add_paragraph(style="List Number")
            inline(p, re.sub(r"^\d+\. ", "", line), size=9.1)
        elif line.startswith("> "):
            p = doc.add_paragraph()
            p.paragraph_format.left_indent = Inches(0.15)
            p.paragraph_format.space_after = Pt(4)
            inline(p, line[2:], size=10.2)
            for run in p.runs:
                run.bold = True
        else:
            p = doc.add_paragraph()
            inline(p, line, size=9.5)
        index += 1

    doc.core_properties.title = "ContextRail 產品一頁式說明 v2.3"
    doc.core_properties.subject = "Environment-aware AI Change Assurance for Cloud-native Business Systems"
    doc.core_properties.author = "ContextRail"
    doc.core_properties.keywords = "Environment Topology; Promotion Policy; AI Change Assurance; Gemini; Cloud Run; Go; agent work order; release assurance"
    doc.core_properties.comments = "Generated from canonical Markdown; source SHA256 " + hashlib.sha256(content.encode()).hexdigest()
    output.parent.mkdir(parents=True, exist_ok=True)
    doc.save(output)
    print(f"Generated {output}")
    print(f"Source SHA256 {hashlib.sha256(content.encode()).hexdigest()}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("source", type=Path)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    build(args.source.resolve(), args.output.resolve())
