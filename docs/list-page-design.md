# List Page Design Patterns

This document describes the standard design patterns for list pages in Strata. Use these patterns consistently across all features.

## Page Structure

```html
{{ define "content" }}
<div class="flex flex-col h-full">
  <!-- Page Header -->
  <div class="flex items-center justify-between mb-2">
    <h1 class="text-2xl font-bold text-gray-900 dark:text-gray-100">Page Title</h1>
  </div>

  <!-- Main Content Section -->
  <section class="flex-1 min-w-0 flex flex-col">
    <!-- Optional: Search/Filter Controls -->
    <!-- Optional: Pagination Info -->
    <!-- Table Container -->
  </section>
</div>
<div id="modal-root"></div>
{{ end }}
```

**Key elements:**
- Outer wrapper: `flex flex-col h-full` - enables flex layout for full height
- Section: `flex-1 min-w-0 flex flex-col` - fills remaining space, enables child flex

## Table Container

```html
<div class="p-4 bg-white dark:bg-gray-800 rounded shadow flex-1 mb-2 overflow-auto">
  <table>...</table>
</div>
```

**Classes explained:**
- `p-4` - padding creates visible dark border around content
- `bg-white dark:bg-gray-800` - light/dark background
- `rounded shadow` - rounded corners with subtle shadow
- `flex-1` - fills remaining vertical space
- `mb-2` - margin before footer
- `overflow-auto` - scrollable if content overflows

## Table Structure

```html
<table class="min-w-full text-sm text-left text-gray-700 dark:text-gray-300">
  <thead class="bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 uppercase text-xs">
    <tr class="border-b border-gray-300 dark:border-gray-600">
      <th class="px-2 py-2">Column Name</th>
      <th class="px-2 py-2 text-right">Actions</th>
    </tr>
  </thead>
  <tbody>
    {{ range .Items }}
    <tr class="border-b border-gray-200 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-900/50">
      <td class="px-2 py-2 align-middle">...</td>
    </tr>
    {{ end }}
  </tbody>
</table>
```

**Header row:**
- Background: `bg-gray-100 dark:bg-gray-700`
- Text: `text-gray-600 dark:text-gray-400 uppercase text-xs`
- Border below: `border-b border-gray-300 dark:border-gray-600`

**Body rows:**
- Border below each row: `border-b border-gray-200 dark:border-gray-600`
- Hover effect: `hover:bg-gray-50 dark:hover:bg-gray-900/50`

**Cells:**
- Padding: `px-2 py-2`
- Vertical alignment: `align-middle`
- Actions column: `text-right`

## Badge/Pill Styling

### Role/Category Pills (rounded-full)
```html
<span class="inline-flex items-center px-2 py-1 rounded-full text-xs bg-purple-100 text-purple-800 dark:bg-purple-900/40 dark:text-purple-400">
  admin
</span>
```

### Status Badges (rounded)
```html
<!-- Active/Success -->
<span class="px-2 py-1 text-xs bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400 rounded">Active</span>

<!-- Inactive/Disabled -->
<span class="px-2 py-1 text-xs bg-gray-200 dark:bg-gray-600 text-gray-700 dark:text-gray-300 rounded">Disabled</span>
```

### Type Badges (for categories like info/warning/critical)
```html
<!-- Info (blue) -->
<span class="inline-flex items-center px-2 py-1 rounded-full text-xs bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400">info</span>

<!-- Warning (yellow) -->
<span class="inline-flex items-center px-2 py-1 rounded-full text-xs bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-400">warning</span>

<!-- Critical (red) -->
<span class="inline-flex items-center px-2 py-1 rounded-full text-xs bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400">critical</span>
```

### Neutral Pills (for auth method, etc.)
```html
<span class="inline-flex items-center px-2 py-1 rounded-full text-xs bg-gray-100 text-gray-800 dark:bg-gray-600 dark:text-gray-300">
  password
</span>
```

## Manage Button

```html
<form
  method="get"
  action="/feature/{{ .ID }}/manage_modal"
  hx-get="/feature/{{ .ID }}/manage_modal?return={{ $.CurrentPath | urlquery }}"
  hx-target="#modal-root"
  hx-swap="innerHTML"
>
  <button
    type="submit"
    class="bg-indigo-600 text-white px-2 py-1 rounded text-xs hover:bg-indigo-700"
    title="Manage item"
  >
    Manage
  </button>
</form>
```

## Manage Modal

```html
{{ define "feature/manage_modal" }}
<div class="fixed inset-0 z-50 flex items-center justify-center">
  <!-- Backdrop -->
  <div class="absolute inset-0 bg-black/40"
       onclick="document.getElementById('modal-root').innerHTML=''"></div>

  <!-- Modal Content -->
  <div class="relative bg-white dark:bg-gray-800 rounded-xl shadow border border-gray-300 dark:border-gray-600 max-w-md w-full p-4 space-y-4">
    <h2 class="text-lg font-semibold text-gray-900 dark:text-gray-100">Manage Item</h2>

    <p class="text-sm text-gray-700 dark:text-gray-300">
      Item details here
    </p>

    <div class="flex flex-col gap-3 pt-2">
      <!-- Action Buttons (centered) -->
      <div class="flex justify-center gap-2">
        <a href="..." class="px-3 py-1 bg-indigo-600 text-white rounded text-sm hover:bg-indigo-700">View</a>
        <a href="..." class="px-3 py-1 bg-indigo-600 text-white rounded text-sm hover:bg-indigo-700">Edit</a>
        <form method="POST" action="..." onsubmit="return confirm('Are you sure?');">
          <button type="submit" class="px-3 py-1 bg-red-600 text-white rounded text-sm hover:bg-red-700">Delete</button>
        </form>
      </div>

      <!-- Cancel Button (bottom left) -->
      <div class="flex justify-start">
        <button
          type="button"
          class="px-3 py-1 border rounded text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700"
          onclick="document.getElementById('modal-root').innerHTML=''"
        >
          Cancel
        </button>
      </div>
    </div>
  </div>
</div>
{{ end }}
```

## Search/Filter Controls (Optional)

```html
<form
  hx-get="/feature"
  hx-target="#content"
  hx-swap="innerHTML"
  hx-push-url="true"
  hx-trigger="submit, keyup changed delay:300ms from:#search-input, change from:#filter-select"
  class="bg-white dark:bg-gray-800 rounded shadow p-3 mb-1 flex flex-wrap items-center gap-2"
>
  <input
    id="search-input" name="search" type="text"
    value="{{ .SearchQuery }}"
    placeholder="Search..."
    class="px-3 py-2 border rounded flex-1 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-400 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-100" />

  <select id="filter-select" name="status" class="px-3 py-2 border rounded text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-gray-100">
    <option value="">All</option>
    <option value="active">Active</option>
    <option value="disabled">Disabled</option>
  </select>

  <a href="..." class="px-4 py-2 border rounded text-sm hover:bg-gray-50 dark:hover:bg-gray-700 mr-6">Clear</a>

  <a href="..." class="px-4 py-2 bg-indigo-600 text-white rounded text-sm hover:bg-indigo-700">Add New</a>
</form>
```

## Pagination Info (Optional)

```html
<div class="flex items-center justify-between mb-1">
  <div class="text-gray-600 dark:text-gray-400 text-sm">
    {{ .RangeStart }}–{{ .RangeEnd }} of {{ .Total }} shown
  </div>
  <div class="flex items-center gap-2">
    {{ if .HasPrev }}
      <a class="inline-flex items-center justify-center h-7 leading-none text-xs px-2 border rounded text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 whitespace-nowrap"
         href="...">Prev</a>
    {{ else }}
      <span class="inline-flex items-center justify-center h-7 leading-none text-xs px-2 border rounded text-gray-400 dark:text-gray-500 whitespace-nowrap">Prev</span>
    {{ end }}
    {{ if .HasNext }}
      <a class="...">Next</a>
    {{ else }}
      <span class="...">Next</span>
    {{ end }}
  </div>
</div>
```

## Empty State

```html
<tr>
  <td colspan="6" class="px-2 py-6 text-center text-gray-500 dark:text-gray-400">
    No items found.
  </td>
</tr>
```

Or outside the table:
```html
<p class="text-gray-500 dark:text-gray-400 py-4 text-center">
  No items. <a href="..." class="text-indigo-600 dark:text-indigo-400 hover:underline">Create one now</a>.
</p>
```

## Primary Action Button (Top Right)

For pages without search controls, place the primary action button in the header:

```html
<div class="mb-4 flex items-center justify-between">
  <h1 class="text-2xl font-bold text-gray-900 dark:text-gray-100">Page Title</h1>
  <a href="/feature/new" class="px-3 py-1 text-sm bg-indigo-600 text-white rounded hover:bg-indigo-700">
    New Item
  </a>
</div>
```

## Success/Error Messages

```html
{{ if .Success }}
  <div class="bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 p-2 rounded mb-4">
    {{ .Success }}
  </div>
{{ end }}

{{ if .Error }}
  <div class="bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 p-2 rounded mb-4">
    {{ .Error }}
  </div>
{{ end }}
```

## Color Reference

| Purpose | Light Mode | Dark Mode |
|---------|-----------|-----------|
| Page background | (from layout) | (from layout) |
| Content container | `bg-white` | `bg-gray-800` |
| Table header bg | `bg-gray-100` | `bg-gray-700` |
| Table header text | `text-gray-600` | `text-gray-400` |
| Table body text | `text-gray-700` | `text-gray-300` |
| Row border | `border-gray-200` | `border-gray-600` |
| Header border | `border-gray-300` | `border-gray-600` |
| Row hover | `bg-gray-50` | `bg-gray-900/50` |
| Primary button | `bg-indigo-600` | `bg-indigo-600` |
| Danger button | `bg-red-600` | `bg-red-600` |
