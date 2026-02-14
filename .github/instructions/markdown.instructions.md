---
description: 'Markdown formatting best practices based on markdownlint rules'
applyTo: '**/*.md'
---

# Markdown Guidelines

Instructions for writing clean, consistent, and accessible Markdown documents based on [markdownlint rules](https://github.com/DavidAnson/markdownlint/blob/main/doc/Rules.md).

## Headings

- **MD001**: Heading levels should only increment by one level at a time (e.g., don't skip from `#` to `###`)
- **MD003**: Use a consistent heading style throughout the document (atx `#` style recommended)
- **MD018**: Include a space after the hash character in atx-style headings (`# Heading` not `#Heading`)
- **MD019**: Use only one space after hash characters in atx-style headings
- **MD020**: Include spaces inside hashes on closed atx-style headings if using that style
- **MD021**: Use only one space inside hashes on closed atx-style headings
- **MD022**: Surround headings with blank lines (one before and one after)
- **MD023**: Headings must start at the beginning of the line without indentation
- **MD024**: Avoid multiple headings with the same content (can be configured to only check sibling headings)
- **MD025**: Use only one top-level heading (`# Title`) per document
- **MD026**: Avoid trailing punctuation in headings (periods, colons, exclamation marks)
- **MD036**: Use headings instead of emphasis (bold/italic) to denote document sections
- **MD041**: Start the document with a top-level heading as the first line
- **MD043**: Follow required heading structure when enforced by project conventions

## Lists

- **MD004**: Use a consistent unordered list style throughout the document (dashes `-` recommended)
- **MD005**: Use consistent indentation for list items at the same level
- **MD007**: Indent nested unordered list items by 2 spaces
- **MD029**: Use consistent ordered list item prefixes (`1.` for all items or sequential numbering)
- **MD030**: Use one space after list markers (`-`, `*`, `1.`)
- **MD032**: Surround lists with blank lines

## Whitespace and Line Length

- **MD009**: Remove trailing spaces at the end of lines (except for intentional line breaks using 2 spaces)
- **MD010**: Use spaces instead of hard tab characters for indentation
- **MD012**: Avoid multiple consecutive blank lines
- **MD013**: Line length — **always ignore this rule**. Do not enforce or warn on line length
- **MD047**: End files with a single newline character

## Code Blocks

- **MD014**: Don't prefix every command with `$` in code blocks unless showing output
- **MD031**: Surround fenced code blocks with blank lines
- **MD038**: Avoid unnecessary leading/trailing spaces inside code span elements
- **MD040**: Specify a language for fenced code blocks for syntax highlighting
- **MD046**: Use a consistent code block style (fenced recommended over indented)
- **MD048**: Use a consistent code fence style (backticks recommended over tildes)

## Links and Images

- **MD011**: Use correct link syntax `[text](https://example.com)` not reversed `(text)[url]`
- **MD034**: Use angle brackets around bare URLs (`<https://example.com>`) or proper link syntax
- **MD039**: Avoid spaces inside link text brackets
- **MD042**: Don't use empty links with no destination
- **MD045**: Include alternate text (alt text) for all images for accessibility
- **MD051**: Ensure link fragments reference valid heading anchors within the document
- **MD052**: Ensure reference links use labels that are defined in the document
- **MD053**: Remove unused link and image reference definitions
- **MD054**: Use consistent link and image styles throughout the document
- **MD059**: Use descriptive link text instead of generic phrases like "click here" or "link"

## Blockquotes

- **MD027**: Use only one space after the blockquote symbol (`>`)
- **MD028**: Avoid blank lines inside blockquotes (use `>` on empty lines to continue the quote)

## Emphasis and Formatting

- **MD037**: Don't include spaces inside emphasis markers (`**bold**` not `** bold **`)
- **MD049**: Use a consistent emphasis style (asterisks `*italic*` recommended)
- **MD050**: Use a consistent strong/bold style (asterisks `**bold**` recommended)

## Tables

- **MD055**: Use consistent leading and trailing pipe characters in table rows
- **MD056**: Ensure all table rows have the same number of columns
- **MD058**: Surround tables with blank lines
- **MD060**: Use consistent table column alignment and formatting style

  Incorrect (no spaces around dashes):

  ```markdown
  | Header 1 | Header 2 |
  |-------|------|
  | Cell 1 | Cell 2 |
  ```

  Correct (spaces around dashes):

  ```markdown
  | Header 1 | Header 2 |
  | ------- | ------ |
  | Cell 1 | Cell 2 |
  ```

## Horizontal Rules

- **MD035**: Use a consistent horizontal rule style throughout the document (e.g., `---`)

## HTML

- **MD033**: Avoid inline HTML when possible; use Markdown syntax instead

## Spelling and Capitalization

- **MD044**: Use correct capitalization for proper names and project terms

## Linting

If the `markdownlint` CLI is available, run it to check for issues. Use the `--fix`
parameter to auto-fix problems and always disable MD013 (line-length).

Example:

```bash
markdownlint --fix --disable MD013 -- path/to/file.md
```