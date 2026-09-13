#!/usr/bin/env python3
"""Build the current ContextRail UI architecture PDF.

This is a deliberately static, vector-first architecture sheet. It documents
the P0 interaction model without implying that the mockup is already backed by
the production API or deployment automation.
"""

from __future__ import annotations

import os
from pathlib import Path
from typing import Iterable, Sequence

from reportlab.lib.colors import HexColor, white
from reportlab.lib.pagesizes import A4, landscape
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from reportlab.pdfgen import canvas


ROOT = Path(__file__).resolve().parents[1]
OUTPUT = ROOT / "output" / "pdf" / "context-rail-ui-architecture.zh-TW.pdf"
PAGE_W, PAGE_H = landscape(A4)

INK = HexColor("#172033")
MUTED = HexColor("#6D7890")
SUBTLE = HexColor("#98A2B5")
LINE = HexColor("#DCE3EC")
CANVAS = HexColor("#F1F4F8")
SURFACE = HexColor("#FFFFFF")
SURFACE_2 = HexColor("#F7F9FC")
BLUE = HexColor("#315EFB")
BLUE_SOFT = HexColor("#EDF2FF")
TEAL = HexColor("#0F9F8D")
TEAL_SOFT = HexColor("#E9F8F5")
AMBER = HexColor("#C47712")
AMBER_SOFT = HexColor("#FFF5E5")
RED = HexColor("#C44949")
RED_SOFT = HexColor("#FFF0F0")
PURPLE = HexColor("#7955D6")
PURPLE_SOFT = HexColor("#F1EDFF")
NAVY = HexColor("#111827")


def register_fonts() -> tuple[str, str]:
    """Register a Chinese-capable font, preferring the macOS UI font."""

    candidates = [
        ("/System/Library/Fonts/STHeiti Light.ttc", "/System/Library/Fonts/STHeiti Medium.ttc"),
        ("/System/Library/Fonts/Hiragino Sans GB.ttc", "/System/Library/Fonts/Hiragino Sans GB.ttc"),
    ]
    for regular_path, medium_path in candidates:
        if not Path(regular_path).exists():
            continue
        try:
            pdfmetrics.registerFont(TTFont("HG-Regular", regular_path, subfontIndex=0))
            pdfmetrics.registerFont(TTFont("HG-Medium", medium_path, subfontIndex=0))
            return "HG-Regular", "HG-Medium"
        except Exception:
            continue
    # The bundled/runtime environment generally has this fallback.
    fallback = "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
    if Path(fallback).exists():
        pdfmetrics.registerFont(TTFont("HG-Regular", fallback))
        pdfmetrics.registerFont(TTFont("HG-Medium", fallback))
        return "HG-Regular", "HG-Medium"
    return "Helvetica", "Helvetica-Bold"


REGULAR, MEDIUM = register_fonts()


def sw(text: str, font: str, size: float) -> float:
    return pdfmetrics.stringWidth(str(text), font, size)


def wrap_text(text: str, font: str, size: float, max_width: float) -> list[str]:
    """Wrap at character boundaries so Chinese and Latin both remain safe."""

    lines: list[str] = []
    for paragraph in str(text).split("\n"):
        current = ""
        for char in paragraph:
            candidate = current + char
            if current and sw(candidate, font, size) > max_width:
                lines.append(current)
                current = char
            else:
                current = candidate
        lines.append(current)
    return lines or [""]


def text(
    c: canvas.Canvas,
    value: str,
    x: float,
    top: float,
    width: float,
    *,
    font: str = REGULAR,
    size: float = 8,
    leading: float | None = None,
    color=INK,
    max_lines: int | None = None,
) -> float:
    leading = leading or size * 1.35
    lines = wrap_text(value, font, size, width)
    if max_lines is not None:
        lines = lines[:max_lines]
    c.setFont(font, size)
    c.setFillColor(color)
    for index, line in enumerate(lines):
        c.drawString(x, top - index * leading, line)
    return top - len(lines) * leading


def centered(c: canvas.Canvas, value: str, x: float, y: float, width: float, *, font=REGULAR, size=8, color=INK) -> None:
    c.setFont(font, size)
    c.setFillColor(color)
    c.drawCentredString(x + width / 2, y, value)


def rounded(c: canvas.Canvas, x: float, y: float, w: float, h: float, fill=SURFACE, stroke=LINE, radius=9, line_width=0.8) -> None:
    c.setLineWidth(line_width)
    c.setStrokeColor(stroke)
    c.setFillColor(fill)
    c.roundRect(x, y, w, h, radius, stroke=1, fill=1)


def pill(c: canvas.Canvas, x: float, y: float, w: float, h: float, label: str, fill, fg=INK, *, size=6.6) -> None:
    c.setFillColor(fill)
    c.setStrokeColor(fill)
    c.roundRect(x, y, w, h, h / 2, stroke=0, fill=1)
    centered(c, label, x, y + h / 2 - size * 0.34, w, font=MEDIUM, size=size, color=fg)


def dot(c: canvas.Canvas, x: float, y: float, color, radius=4) -> None:
    c.setFillColor(color)
    c.setStrokeColor(color)
    c.circle(x, y, radius, stroke=0, fill=1)


def line(c: canvas.Canvas, x1: float, y1: float, x2: float, y2: float, color=LINE, width=0.8) -> None:
    c.setStrokeColor(color)
    c.setLineWidth(width)
    c.line(x1, y1, x2, y2)


def page_header(c: canvas.Canvas, number: str, section: str, title: str, subtitle: str) -> None:
    c.setFillColor(BLUE)
    c.rect(0, PAGE_H - 5, PAGE_W, 5, stroke=0, fill=1)
    text(c, "CONTEXTRAIL", 62, PAGE_H - 63, 180, font=MEDIUM, size=8.5, color=BLUE)
    text(c, section.upper(), PAGE_W - 270, PAGE_H - 63, 208, font=REGULAR, size=7.2, color=MUTED, max_lines=1)
    c.setFillColor(BLUE)
    c.circle(76, PAGE_H - 104, 14, stroke=0, fill=1)
    centered(c, number, 62, PAGE_H - 107.5, 28, font=MEDIUM, size=8, color=white)
    text(c, title, 102, PAGE_H - 94, 610, font=MEDIUM, size=21, leading=23, color=INK)
    text(c, subtitle, 102, PAGE_H - 123, 650, font=REGULAR, size=9, leading=12, color=MUTED)


def footer(c: canvas.Canvas, page: int) -> None:
    line(c, 62, 40, PAGE_W - 62, 40, LINE, 0.7)
    text(c, "UI architecture concept · Project-based Cloud Run workflow", 62, 25, 360, size=7.2, color=SUBTLE)
    right = f"2026-09-12 · {page}/4"
    c.setFont(REGULAR, 7.2)
    c.setFillColor(SUBTLE)
    c.drawRightString(PAGE_W - 62, 25, right)


def nav_sidebar(c: canvas.Canvas, x: float, y: float, w: float, h: float, selected: str = "總覽") -> None:
    c.setFillColor(NAVY)
    c.roundRect(x, y, w, h, 10, stroke=0, fill=1)
    c.setFillColor(white)
    c.setFont(MEDIUM, 12)
    c.drawString(x + 15, y + h - 25, "C")
    text(c, "ContextRail", x + 34, y + h - 22, w - 45, font=MEDIUM, size=6.5, color=white, max_lines=1)
    text(c, "AI-native control plane", x + 34, y + h - 36, w - 45, size=5.1, color=SUBTLE, max_lines=1)
    c.setFillColor(HexColor("#1B2638"))
    c.roundRect(x + 9, y + h - 78, w - 18, 28, 7, stroke=0, fill=1)
    dot(c, x + 20, y + h - 64, HexColor("#7786FF"), 3.5)
    text(c, "Internal Approval", x + 30, y + h - 59, w - 40, font=MEDIUM, size=5.1, color=white, max_lines=1)
    text(c, "Project · 4 envs · staging", x + 30, y + h - 70, w - 40, size=4.3, color=SUBTLE, max_lines=1)
    text(c, "PORTFOLIO", x + 11, y + h - 97, w - 20, font=MEDIUM, size=5.5, color=SUBTLE)
    items = ["專案管理", "總覽", "需求變更", "架構與影響", "證據與審查", "角色報告", "Project 設定", "Policy gate", "Audit trail"]
    positions = [y + h - 119 - i * 21 for i in range(len(items))]
    for label, item_y in zip(items, positions):
        if label == selected:
            c.setFillColor(HexColor("#30447F"))
            c.roundRect(x + 8, item_y - 6, w - 16, 17, 5, stroke=0, fill=1)
        text(c, label, x + 19, item_y + 1, w - 30, font=MEDIUM if label == selected else REGULAR, size=6.2, color=white if label == selected else HexColor("#D5DBEA"))


def mini_env(c: canvas.Canvas, x: float, y: float, w: float, label: str, sub: str, fill, fg=INK) -> None:
    rounded(c, x, y, w, 28, fill, fill, 6, 0.5)
    text(c, label, x + 8, y + 17, w - 16, font=MEDIUM, size=6.2, color=fg)
    text(c, sub, x + 8, y + 7, w - 16, size=5.2, color=fg)


def draw_registry(c: canvas.Canvas, x: float, y: float, w: float, h: float) -> None:
    rounded(c, x, y, w, h, SURFACE, LINE, 10)
    text(c, "PROJECT REGISTRY", x + 14, y + h - 21, w - 28, font=MEDIUM, size=7.2, color=BLUE)
    text(c, "Project 是需求、環境、政策、文件與交付證據的治理邊界。", x + 14, y + h - 35, w - 28, size=6.1, color=MUTED)
    pill(c, x + w - 94, y + h - 31, 80, 14, "SSOT BOUNDARY", TEAL_SOFT, TEAL, size=5.2)
    summary_y = y + h - 72
    for index, (label, value, fill, fg) in enumerate([
        ("Active", "2", BLUE_SOFT, BLUE),
        ("Attention", "1", AMBER_SOFT, AMBER),
        ("Archived", "1", SURFACE_2, MUTED),
    ]):
        sx = x + 14 + index * ((w - 28) / 3)
        swidth = (w - 38) / 3
        rounded(c, sx, summary_y, swidth, 28, fill, fill, 5, 0)
        text(c, label, sx + 7, summary_y + 19, swidth - 14, size=5.2, color=MUTED)
        text(c, value, sx + 7, summary_y + 8, swidth - 14, font=MEDIUM, size=10, color=fg)
    table_top = summary_y - 20
    columns = [28, 86, 47, 45]
    headers = ["KEY", "PROJECT", "STATUS", "ACTION"]
    xpos = x + 14
    for header, col in zip(headers, columns):
        text(c, header, xpos, table_top, col, font=MEDIUM, size=5.3, color=SUBTLE)
        xpos += col
    line(c, x + 14, table_top - 6, x + w - 14, table_top - 6, LINE, 0.6)
    rows = [
        ("APV", "Internal Approval", "ACTIVE", "進入 Workspace", BLUE, BLUE_SOFT),
        ("BIL", "Billing Portal", "ACTIVE", "進入 Workspace", TEAL, TEAL_SOFT),
        ("LEG", "Legacy Claims", "ARCHIVED", "查看歷史", MUTED, SURFACE_2),
    ]
    for index, (key, project, status, action, fg, fill) in enumerate(rows):
        row_top = table_top - 19 - index * 38
        if index == 0:
            rounded(c, x + 10, row_top - 25, w - 20, 31, BLUE_SOFT, BLUE_SOFT, 5, 0)
        c.setFillColor(fill)
        c.roundRect(x + 17, row_top - 18, 22, 14, 4, stroke=0, fill=1)
        centered(c, key, x + 17, row_top - 13.5, 22, font=MEDIUM, size=5.5, color=fg)
        text(c, project, x + 47, row_top - 7, 78, font=MEDIUM, size=5.6, color=INK)
        pill(c, x + 131, row_top - 16, 42, 12, status, fill, fg, size=4.7)
        text(c, action, x + 181, row_top - 7, 54, font=MEDIUM, size=5.2, color=BLUE if index < 2 else MUTED)
        line(c, x + 14, row_top - 29, x + w - 14, row_top - 29, LINE, 0.45)
    text(c, "Active Project → 進入 Workspace；Archived → 只保留歷史查詢。", x + 14, y + 16, w - 28, font=MEDIUM, size=5.6, color=MUTED)


def draw_workspace(c: canvas.Canvas, x: float, y: float, w: float, h: float) -> None:
    rounded(c, x, y, w, h, SURFACE, LINE, 10)
    side_w = 95
    nav_sidebar(c, x, y, side_w, h, "總覽")
    cx = x + side_w
    c.setFillColor(SURFACE_2)
    c.rect(cx, y, w - side_w, h, stroke=0, fill=1)
    line(c, cx, y + h - 32, x + w, y + h - 32, LINE, 0.6)
    text(c, "Projects / Internal Approval System", cx + 14, y + h - 20, 210, size=5.8, color=MUTED)
    pill(c, x + w - 108, y + h - 27, 94, 15, "staging · Cloud Run", TEAL_SOFT, TEAL, size=5.1)
    text(c, "PROJECT DECISION WORKSPACE", cx + 15, y + h - 51, 220, font=MEDIUM, size=5.2, color=BLUE)
    text(c, "CHG-2026-0017 · 新增申請附件上傳", cx + 15, y + h - 67, 255, font=MEDIUM, size=11, color=INK)
    pill(c, x + w - 103, y + h - 76, 89, 16, "NEEDS_REVIEW", AMBER_SOFT, AMBER, size=5.4)
    text(c, "把需求、架構影響、PR / review 證據與 promotion gate 放在同一個決策脈絡裡。", cx + 15, y + h - 84, 285, size=5.4, color=MUTED)
    top = y + h - 111
    rounded(c, cx + 15, top - 29, w - side_w - 29, 29, SURFACE, LINE, 7)
    steps = ["需求確認", "架構決策", "批准開發", "PR / Review", "Staging gate", "Verified"]
    step_w = (w - side_w - 47) / 6
    for idx, label in enumerate(steps):
        sx = cx + 20 + idx * step_w
        c.setFillColor(TEAL_SOFT if idx == 0 else BLUE_SOFT if idx == 1 else HexColor("#EEF1F6"))
        c.circle(sx + 7, top - 14, 7, stroke=0, fill=1)
        centered(c, str(idx + 1), sx, top - 16.3, 14, font=MEDIUM, size=5.2, color=BLUE if idx > 1 else TEAL)
        text(c, label, sx + 17, top - 12, step_w - 18, size=4.8, color=BLUE if idx == 1 else MUTED)
        if idx < 5:
            line(c, sx + 23, top - 14, sx + step_w - 3, top - 14, LINE, 0.55)
    topo_y = y + h - 190
    rounded(c, cx + 15, topo_y, w - side_w - 29, 57, SURFACE, LINE, 8)
    text(c, "PROJECT ENVIRONMENT TOPOLOGY", cx + 25, topo_y + 43, 190, font=MEDIUM, size=5.1, color=BLUE)
    mini_w = (w - side_w - 66) / 4
    envs = [("Development", "branch", TEAL_SOFT), ("Testing", "CI evidence", TEAL_SOFT), ("Staging", "2 gaps", BLUE_SOFT), ("Production", "blocked", RED_SOFT)]
    for idx, (label, sub, fill) in enumerate(envs):
        mini_env(c, cx + 25 + idx * (mini_w + 6), topo_y + 9, mini_w, label, sub, fill, TEAL if idx < 2 else BLUE if idx == 2 else RED)
    card_y = y + 20
    card_h = 104
    left_w = (w - side_w - 44) * 0.62
    right_w = (w - side_w - 44) - left_w - 9
    rounded(c, cx + 15, card_y, left_w, card_h, SURFACE, BLUE_SOFT, 8, 1)
    text(c, "Decision record · DR-2026-0084 · v3", cx + 27, card_y + card_h - 18, left_w - 40, size=5.1, color=MUTED)
    text(c, "目前可以進入哪個環境？", cx + 27, card_y + card_h - 37, left_w - 95, font=MEDIUM, size=9, color=INK)
    text(c, "建議先在 staging 驗證附件流程；Production gate 仍保持 blocked。", cx + 27, card_y + card_h - 56, left_w - 38, size=5.5, color=MUTED, max_lines=2)
    text(c, "Recommendation: 可進 staging，暫不允許 production", cx + 27, card_y + 19, left_w - 38, font=MEDIUM, size=5.5, color=BLUE)
    rounded(c, cx + 15 + left_w + 9, card_y, right_w, card_h, SURFACE, BLUE_SOFT, 8, 1)
    text(c, "Decision gate", cx + 30 + left_w, card_y + card_h - 21, right_w - 22, font=MEDIUM, size=7.4, color=INK)
    pill(c, cx + 15 + left_w + right_w - 28, card_y + card_h - 28, 35, 15, "STAGING", BLUE_SOFT, BLUE, size=4.5)
    for idx, label in enumerate(["Topology 有版本", "PR / CI evidence", "RTO / RPO 待確認", "Production 未開放"]):
        col = TEAL if idx < 2 else AMBER if idx == 2 else RED
        dot(c, cx + 32 + left_w, card_y + 64 - idx * 14, col, 3)
        text(c, label, cx + 41 + left_w, card_y + 66 - idx * 14, right_w - 49, size=5.1, color=MUTED)


def page_one(c: canvas.Canvas) -> None:
    page_header(c, "01", "Project entry · workspace", "從 Project 進入 Workspace", "Project 是治理邊界；Workspace 才是每張 Ticket 的決策操作面。")
    route_y = 425
    rounded(c, 62, route_y, PAGE_W - 124, 38, BLUE_SOFT, BLUE_SOFT, 8, 0)
    route = [("1", "Project Registry", "找到 active Project"), ("2", "Project Settings", "確認 contract 與版本"), ("3", "Project Workspace", "處理 Ticket 與 gate")]
    route_w = (PAGE_W - 156) / 3
    for idx, (number, label, sub) in enumerate(route):
        sx = 78 + idx * route_w
        c.setFillColor(BLUE if idx == 2 else SURFACE)
        c.circle(sx + 9, route_y + 19, 9, stroke=0, fill=1)
        centered(c, number, sx, route_y + 16, 18, font=MEDIUM, size=6.1, color=white if idx == 2 else BLUE)
        text(c, label, sx + 25, route_y + 23, route_w - 45, font=MEDIUM, size=6.9, color=BLUE if idx == 2 else INK)
        text(c, sub, sx + 25, route_y + 11, route_w - 45, size=5.5, color=MUTED)
        if idx < 2:
            line(c, sx + route_w - 12, route_y + 19, sx + route_w + 6, route_y + 19, BLUE, 0.8)
    draw_registry(c, 62, 86, 238, 319)
    draw_workspace(c, 320, 86, PAGE_W - 382, 319)
    rounded(c, 62, 56, PAGE_W - 124, 21, AMBER_SOFT, AMBER_SOFT, 6, 0)
    text(c, "互動語義", 76, 69, 64, font=MEDIUM, size=5.8, color=AMBER)
    text(c, "Active Project 的進入動作會帶入 project_id；所有 Ticket、DecisionRecord、Evidence 與 Receipt 由同一個 SSOT 串起來。", 145, 69, PAGE_W - 225, size=5.8, color=MUTED)
    footer(c, 1)


def state_card(c: canvas.Canvas, x: float, y: float, w: float, h: float, label: str, description: str, fill, fg) -> None:
    rounded(c, x, y, w, h, fill, fill, 7, 0)
    text(c, label, x + 10, y + h - 16, w - 20, font=MEDIUM, size=7.2, color=fg)
    text(c, description, x + 10, y + h - 32, w - 20, size=5.7, color=MUTED, max_lines=2)


def table_grid(c: canvas.Canvas, x: float, y: float, widths: Sequence[float], row_heights: Sequence[float], headers: Sequence[str], rows: Sequence[Sequence[str]], header_fill=BLUE_SOFT) -> None:
    total_w = sum(widths)
    total_h = sum(row_heights)
    rounded(c, x, y, total_w, total_h, SURFACE, LINE, 9, 0.8)
    c.setFillColor(header_fill)
    c.roundRect(x, y + total_h - row_heights[0], total_w, row_heights[0], 9, stroke=0, fill=1)
    # Mask the lower corners of the rounded header so the table remains rectangular.
    c.rect(x, y + total_h - row_heights[0], total_w, row_heights[0] - 9, stroke=0, fill=1)
    xpos = x
    for header, width in zip(headers, widths):
        text(c, header, xpos + 10, y + total_h - 16, width - 18, font=MEDIUM, size=6.3, color=MUTED)
        xpos += width
    cursor_y = y + total_h - row_heights[0]
    for row_index, (row, row_h) in enumerate(zip(rows, row_heights[1:])):
        line(c, x, cursor_y, x + total_w, cursor_y, LINE, 0.55)
        xpos = x
        for value, width in zip(row, widths):
            text(c, value, xpos + 10, cursor_y - 15, width - 18, font=MEDIUM if row_index == 0 else REGULAR, size=6.2, leading=8, color=INK if row_index == 0 else MUTED, max_lines=2)
            xpos += width
        cursor_y -= row_h
    xpos = x
    for width in widths[:-1]:
        xpos += width
        line(c, xpos, y, xpos, y + total_h, LINE, 0.45)


def page_two(c: canvas.Canvas) -> None:
    page_header(c, "02", "Project CRUD · settings", "Project CRUD 與狀態語義", "Project 設定不是一般後台表單；每次變更都會影響後續決策是否仍然有效。")
    state_y = 402
    state_w = (PAGE_W - 124 - 18) / 4
    states = [
        ("DRAFT", "尚未成為 active 邊界", SURFACE_2, MUTED),
        ("ACTIVE", "可建立 Ticket 與 promotion", TEAL_SOFT, TEAL),
        ("STALE", "受影響判斷要重新確認", AMBER_SOFT, AMBER),
        ("ARCHIVED", "停止新變更，保留歷史證據", RED_SOFT, RED),
    ]
    for idx, (label, desc, fill, fg) in enumerate(states):
        state_card(c, 62 + idx * (state_w + 6), state_y, state_w, 48, label, desc, fill, fg)
    text(c, "CRUD 的核心不是按鈕，而是狀態轉換與可追溯後果", 62, 374, 400, font=MEDIUM, size=9, color=INK)
    left_x, right_x, panel_y, panel_h = 62, 430, 135, 210
    rounded(c, left_x, panel_y, 340, panel_h, SURFACE, LINE, 10)
    text(c, "操作與後果", left_x + 15, panel_y + panel_h - 22, 160, font=MEDIUM, size=8.5, color=BLUE)
    text(c, "每個操作只改 Project 邊界，不直接替使用者執行 production。", left_x + 15, panel_y + panel_h - 37, 300, size=5.9, color=MUTED)
    table_grid(
        c,
        left_x + 14,
        panel_y + 10,
        [70, 91, 151],
        [25, 38, 38, 38, 38],
        ["ACTION", "STATE", "VISIBLE EFFECT"],
        [
            ["Create", "DRAFT → ACTIVE", "建立 identity、repo、owner、topology"],
            ["Read", "ACTIVE / ARCHIVED", "依 membership 查詢相應證據"],
            ["Update", "ACTIVE → v+1", "受影響 Decision / Approval → STALE"],
            ["Archive", "ACTIVE → ARCHIVED", "停止新 Ticket / promotion，保留 receipt"],
        ],
    )
    rounded(c, right_x, panel_y, 350, panel_h, SURFACE, LINE, 10)
    text(c, "PROJECT SETTINGS", right_x + 15, panel_y + panel_h - 22, 160, font=MEDIUM, size=8.5, color=BLUE)
    pill(c, right_x + 260, panel_y + panel_h - 31, 69, 15, "v3 ACTIVE", BLUE_SOFT, BLUE, size=5.2)
    text(c, "Project identity", right_x + 15, panel_y + panel_h - 47, 140, font=MEDIUM, size=7.5, color=INK)
    fields = [("Project key", "APV"), ("Project name", "Internal Approval System"), ("Repository", "github.com/team/internal-approval"), ("Technical owner", "William Chiu"), ("Business owner", "Product Operations"), ("Data classification", "Internal")]
    field_w = 145
    for idx, (label, value) in enumerate(fields):
        col = idx % 2
        row = idx // 2
        fx = right_x + 15 + col * (field_w + 14)
        fy = panel_y + panel_h - 78 - row * 37
        text(c, label, fx, fy + 17, field_w, font=MEDIUM, size=5.2, color=MUTED)
        rounded(c, fx, fy, field_w, 17, SURFACE_2, LINE, 4, 0.6)
        text(c, value, fx + 7, fy + 11, field_w - 14, size=5.5, color=INK, max_lines=1)
    rounded(c, right_x + 15, panel_y + 17, 320, 23, AMBER_SOFT, AMBER_SOFT, 5, 0)
    text(c, "儲存後：寫入 Audit trail；受影響判斷標記 STALE。", right_x + 24, panel_y + 31, 300, font=MEDIUM, size=5.6, color=AMBER)
    # Version flow
    flow_y = 78
    flow = [("Project settings", BLUE_SOFT, BLUE), ("Version +1", PURPLE_SOFT, PURPLE), ("Decision stale", AMBER_SOFT, AMBER), ("Human re-review", TEAL_SOFT, TEAL)]
    fw = (PAGE_W - 124 - 18) / 4
    for idx, (label, fill, fg) in enumerate(flow):
        fx = 62 + idx * (fw + 6)
        rounded(c, fx, flow_y, fw, 36, fill, fill, 7, 0)
        centered(c, label, fx, flow_y + 14, fw, font=MEDIUM, size=6.4, color=fg)
        if idx < 3:
            line(c, fx + fw + 1, flow_y + 18, fx + fw + 5, flow_y + 18, fg, 1)
    footer(c, 2)


def page_three(c: canvas.Canvas) -> None:
    page_header(c, "03", "Workspace · decision lifecycle", "Workspace 互動與決策生命週期", "所有可執行動作都先經過資料完整性、架構影響與人類 gate；不足資料時明確請求補件。")
    steps = ["需求確認", "架構決策", "批准開發", "PR / Review", "Staging gate", "Verified"]
    sx = 62
    step_w = (PAGE_W - 124 - 25) / 6
    for idx, label in enumerate(steps):
        c.setFillColor(TEAL_SOFT if idx == 0 else BLUE_SOFT if idx == 1 else SURFACE_2)
        c.setStrokeColor(TEAL if idx == 0 else BLUE if idx == 1 else LINE)
        c.setLineWidth(0.8)
        c.roundRect(sx, 420, step_w, 34, 7, stroke=1, fill=1)
        c.setFillColor(TEAL if idx == 0 else BLUE if idx == 1 else SUBTLE)
        c.circle(sx + 13, 437, 8, stroke=0, fill=1)
        centered(c, str(idx + 1), sx + 5, 434.5, 16, font=MEDIUM, size=5.8, color=white if idx < 2 else INK)
        text(c, label, sx + 27, 440, step_w - 34, font=MEDIUM, size=6, color=BLUE if idx < 2 else MUTED)
        if idx < 5:
            line(c, sx + step_w + 1, 437, sx + step_w + 5, 437, LINE, 1)
        sx += step_w + 5
    text(c, "P0 的五個決策面", 62, 389, 300, font=MEDIUM, size=9, color=INK)
    cards = [
        ("1  Project contract", "環境順序、target、promotion policy 與 project_id。", BLUE, BLUE_SOFT),
        ("2  Ticket context", "需求、變更來源、branch / PR 與驗收條件。", PURPLE, PURPLE_SOFT),
        ("3  Impact analysis", "服務、資料、權限、Cloud Run 路徑與成本假設。", TEAL, TEAL_SOFT),
        ("4  Evidence", "PR、CI、review、deployment 與 receipt 的證據鏈。", AMBER, AMBER_SOFT),
        ("5  Human gate", "人確認後才建立 Work Order；production 預設阻擋。", RED, RED_SOFT),
    ]
    card_w = (PAGE_W - 124 - 24) / 5
    for idx, (label, desc, fg, fill) in enumerate(cards):
        cx = 62 + idx * (card_w + 6)
        rounded(c, cx, 282, card_w, 91, fill, fg, 8, 0.8)
        text(c, label, cx + 10, 351, card_w - 20, font=MEDIUM, size=7, color=fg)
        text(c, desc, cx + 10, 331, card_w - 20, size=5.8, leading=8, color=MUTED, max_lines=4)
    text(c, "核心狀態", 62, 246, 120, font=MEDIUM, size=9, color=INK)
    state_flow = [
        ("DRAFT", "草稿", SURFACE_2, MUTED),
        ("NEEDS_INPUT", "資料不足", AMBER_SOFT, AMBER),
        ("READY_FOR_DECISION", "可決策", BLUE_SOFT, BLUE),
        ("APPROVED_FOR_STAGING", "允許 staging", TEAL_SOFT, TEAL),
        ("STAGING_VERIFIED", "證據完成", BLUE_SOFT, BLUE),
        ("PRODUCTION_BLOCKED", "尚不可上線", RED_SOFT, RED),
    ]
    state_x = 62
    state_w = (PAGE_W - 124 - 25) / 6
    for idx, (label, desc, fill, fg) in enumerate(state_flow):
        pill(c, state_x, 207, state_w, 20, label, fill, fg, size=5.2)
        centered(c, desc, state_x, 193, state_w, size=5.7, color=MUTED)
        if idx < 5:
            line(c, state_x + state_w + 1, 217, state_x + state_w + 4, 217, SUBTLE, 0.8)
        state_x += state_w + 5
    rounded(c, 62, 75, PAGE_W - 124, 95, SURFACE, LINE, 9)
    text(c, "可點擊動作的預期回饋", 78, 150, 210, font=MEDIUM, size=8, color=BLUE)
    action_data = [
        ("匯出角色報告", "產出 PM / 主管 / 技術 / Agent 可讀的附件。", BLUE),
        ("請求補充資料", "建立明確補件清單；AI 不猜測缺少的事實。", AMBER),
        ("建立 Work Order", "先顯示 gate 預覽；人確認後才觸發執行。", TEAL),
    ]
    col_w = (PAGE_W - 156) / 3
    for idx, (label, desc, fg) in enumerate(action_data):
        ax = 78 + idx * col_w
        dot(c, ax, 122, fg, 4)
        text(c, label, ax + 12, 126, col_w - 18, font=MEDIUM, size=6.7, color=fg)
        text(c, desc, ax + 12, 110, col_w - 18, size=5.8, color=MUTED, max_lines=2)
    footer(c, 3)


def page_four(c: canvas.Canvas) -> None:
    page_header(c, "04", "Role lens · first build slice", "角色報告與第一個開發切片", "同一份 Single Source of Truth 依角色改變閱讀方式；報告是附件，決策狀態才是產品核心。")
    text(c, "Role lens", 62, 442, 140, font=MEDIUM, size=9, color=INK)
    text(c, "不同身份回答不同問題，但不複製互相矛盾的資料。", 158, 442, 330, size=6.3, color=MUTED)
    table_grid(
        c,
        62,
        226,
        [112, 165, 177, 129, 139],
        [27, 43, 43, 43, 43, 43],
        ["ROLE", "KEY QUESTION", "REPORT / SCREEN", "TONE", "ACTION"],
        [
            ["PM / 主管", "要不要做？現在能到哪裡？", "Executive Change Decision Brief", "少術語、重影響", "確認優先級、補資料"],
            ["Solution Architect", "影響哪些服務？替代方案？", "Technical Architecture & Cost Report", "假設、取捨、成本", "選方案、確認架構"],
            ["FDE / Tech Lead", "能否交給工程團隊？", "Implementation Readiness Pack", "驗收條件、依賴、缺口", "建立 Work Order"],
            ["DevOps / Release", "證據是否足夠進下一環境？", "Release Receipt", "測試、build、gate", "執行或阻擋 promotion"],
        ],
    )
    text(c, "第一個可驗證的 vertical slice", 62, 195, 260, font=MEDIUM, size=9, color=INK)
    text(c, "先證明一條完整決策鏈，再逐步接上 AINE Enterprise contract。", 250, 195, 410, size=6.2, color=MUTED)
    step_y = 84
    step_w = (PAGE_W - 124 - 18) / 4
    slice_data = [
        ("1", "Freeze topology", "鎖定 Project 的環境、順序、target 與 policy。", BLUE, BLUE_SOFT),
        ("2", "導入 Spec Kit", "只規格化 Project Decision Workspace 的 contract。", PURPLE, PURPLE_SOFT),
        ("3", "Go backend", "fixture-backed record、read-only overview、gate、role lens。", TEAL, TEAL_SOFT),
        ("4", "AINE contract", "以 versioned HTTP contract 接 evidence、policy、approval、audit。", AMBER, AMBER_SOFT),
    ]
    for idx, (number, label, desc, fg, fill) in enumerate(slice_data):
        x = 62 + idx * (step_w + 6)
        rounded(c, x, step_y, step_w, 84, SURFACE, LINE, 8)
        c.setFillColor(fill)
        c.circle(x + 19, step_y + 62, 11, stroke=0, fill=1)
        centered(c, number, x + 8, step_y + 59, 22, font=MEDIUM, size=6.8, color=fg)
        text(c, label, x + 38, step_y + 67, step_w - 48, font=MEDIUM, size=7.2, color=INK)
        text(c, desc, x + 12, step_y + 42, step_w - 24, size=5.8, leading=8, color=MUTED, max_lines=4)
    rounded(c, 62, 53, PAGE_W - 124, 21, AMBER_SOFT, AMBER_SOFT, 6, 0)
    text(c, "P0 邊界", 76, 66, 56, font=MEDIUM, size=5.8, color=AMBER)
    text(c, "不做多租戶 SaaS、任意遠端 agent、完整 FDE 自主代理、內網 connector、Cloud Assist MCP、完整 rollback 與 production readiness。", 135, 66, PAGE_W - 211, size=5.7, color=MUTED)
    footer(c, 4)


def build() -> None:
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    c = canvas.Canvas(str(OUTPUT), pagesize=landscape(A4), pageCompression=1)
    c.setTitle("ContextRail UI Architecture")
    c.setAuthor("ContextRail")
    c.setSubject("Project Registry, Project Workspace and Project Settings interaction architecture")
    c.setKeywords("ContextRail, Project Workspace, Cloud Run, decision gate, UI architecture")
    c.setFillColor(CANVAS)
    for page in (page_one, page_two, page_three, page_four):
        c.setFillColor(CANVAS)
        c.rect(0, 0, PAGE_W, PAGE_H, stroke=0, fill=1)
        page(c)
        c.showPage()
    c.save()
    print(OUTPUT)


if __name__ == "__main__":
    build()
