package main

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
)

// The icons of the folder sidebar: an SVG of 16 by 16 per kind of file, in
// its own colors, which read on both light and dark sidebars. A file's
// name picks its icon (README, Makefile, package.json), else its
// extension, else the plain page. They are drawn here rather than taken
// from an icon theme, and parsed the first time they show, as most
// windows show no folder.

// fileIcons holds the body of each icon's SVG, its viewBox 16 by 16.
var fileIcons = map[string]string{
	"folder":      folderIcon("#3E8DDD", "#66AEF2", false),
	"folder-open": folderIcon("#3E8DDD", "#66AEF2", true),
	// Folders whose names start with a dot, as .git and .github.
	"folder-dot":      folderIcon("#7D8B9C", "#A2AFBE", false),
	"folder-dot-open": folderIcon("#7D8B9C", "#A2AFBE", true),

	"file": page("#9AA6B4", "#CDD5DE", ""),
	"text": page("#9AA6B4", "#CDD5DE",
		`<path d="M5.5 7.6h5M5.5 9.6h5M5.5 11.6h3" stroke="#fff" stroke-width="1.1" stroke-linecap="round"/>`),
	"code": page("#9AA6B4", "#CDD5DE",
		`<path d="M6.6 7.6 5 9.4l1.6 1.8M9.4 7.6 11 9.4l-1.6 1.8" fill="none" stroke="#fff" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/>`),
	"pdf": page("#E5484D", "#F5A9AB",
		`<path d="M5.5 7.6h5M5.5 9.6h5M5.5 11.6h3" stroke="#fff" stroke-width="1.1" stroke-linecap="round"/>`),

	"markdown": `<rect x=".8" y="3.2" width="14.4" height="9.6" rx="2" fill="#3F8BD6"/>` +
		`<path d="M3.4 10.3V5.7l2 2.4 2-2.4v4.6M11.3 5.7v4.4M9.6 8.6l1.7 1.8 1.7-1.8" fill="none" stroke="#fff" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>`,
	"readme": `<circle cx="8" cy="8" r="6.6" fill="#3F8BD6"/><circle cx="8" cy="4.9" r="1.05" fill="#fff"/>` +
		`<path d="M8 7.3v4.3" stroke="#fff" stroke-width="1.7" stroke-linecap="round"/>`,
	"license": `<rect x="1.5" y="2.5" width="13" height="9.2" rx="1.6" fill="#E2A33A"/>` +
		`<path d="M3.9 5.3h5.2M3.9 7.4h3.4" stroke="#fff" stroke-width="1.1" stroke-linecap="round"/>` +
		`<path d="M10.3 10.2 9.7 14l1.8-1 1.8 1-.6-3.8z" fill="#C4433A"/><circle cx="11.5" cy="8.9" r="2.2" fill="#D9534F"/>`,

	"go": `<g fill="none" stroke="#00ADD8" stroke-width="1.8" stroke-linecap="round" transform="translate(1.3 0) skewX(-10)">` +
		`<path d="M6.25 6A2.75 2.75 0 1 0 7.05 8.45H5.3"/><circle cx="11.15" cy="8" r="2.75"/></g>`,
	"js": `<rect x="1.5" y="1.5" width="13" height="13" rx="2.2" fill="#F0D64B"/>` +
		`<path d="M7.2 7.2V11c0 1-.6 1.6-1.5 1.6-.6 0-1.1-.3-1.3-.8M12.5 7.9c-.2-.5-.7-.8-1.3-.8-.8 0-1.3.4-1.3 1 0 1.5 2.8 1 2.8 2.7 0 .7-.6 1.2-1.5 1.2-.7 0-1.3-.3-1.6-.9" fill="none" stroke="#2B2B2B" stroke-width="1.35" stroke-linecap="round" stroke-linejoin="round"/>`,
	"ts": `<rect x="1.5" y="1.5" width="13" height="13" rx="2.2" fill="#3178C6"/>` +
		`<path d="M3.7 7.2h4.2M5.8 7.2v5.4M12.5 7.9c-.2-.5-.7-.8-1.3-.8-.8 0-1.3.4-1.3 1 0 1.5 2.8 1 2.8 2.7 0 .7-.6 1.2-1.5 1.2-.7 0-1.3-.3-1.6-.9" fill="none" stroke="#fff" stroke-width="1.35" stroke-linecap="round" stroke-linejoin="round"/>`,
	"react":    atom("#1FA8D8"),
	"react-ts": atom("#3B82C8"),
	"json": `<path d="M5.6 2.5c-1.4 0-2 .6-2 1.8v1.9c0 .9-.5 1.5-1.4 1.8.9.3 1.4.9 1.4 1.8v1.9c0 1.2.6 1.8 2 1.8M10.4 2.5c1.4 0 2 .6 2 1.8v1.9c0 .9.5 1.5 1.4 1.8-.9.3-1.4.9-1.4 1.8v1.9c0 1.2-.6 1.8-2 1.8" fill="none" stroke="#E5A230" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>` +
		`<circle cx="8" cy="8" r="1" fill="#E5A230"/>`,
	"yaml":   `<path d="M2.2 4.3h1.6M6 4.3h7.8M4.4 8h1.6M8.2 8h5.6M4.4 11.7h1.6M8.2 11.7h4" stroke="#DC5A50" stroke-width="1.7" stroke-linecap="round"/>`,
	"config": gear("#8794A3"),
	"env": `<path d="M2.3 4.3h11.4M2.3 8h11.4M2.3 11.7h11.4" stroke="#D9A43A" stroke-width="1.3" stroke-linecap="round"/>` +
		`<g fill="#D9A43A" stroke="#fff" stroke-width=".9"><circle cx="5.3" cy="4.3" r="1.9"/><circle cx="10.6" cy="8" r="1.9"/><circle cx="7" cy="11.7" r="1.9"/></g>`,
	"xml": `<path d="M4.8 4.5 1.6 8l3.2 3.5M11.2 4.5 14.4 8l-3.2 3.5M9.3 3.3l-2.6 9.4" fill="none" stroke="#EE8A3A" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>`,
	"html": `<path d="M2 1.5h12l-1.1 11.6L8 14.6l-4.9-1.5z" fill="#E5532D"/>` +
		`<path d="M6.4 5.4 4.6 7.8l1.8 2.4M9.6 5.4l1.8 2.4-1.8 2.4" fill="none" stroke="#fff" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>`,
	"css": `<path d="M2 1.5h12l-1.1 11.6L8 14.6l-4.9-1.5z" fill="#2F7FD4"/>` +
		`<path d="M5 6.3h6M4.8 9.3h6M7.1 4.3l-.6 7M9.7 4.3l-.6 7" fill="none" stroke="#fff" stroke-width="1.2" stroke-linecap="round"/>`,
	"sass": `<circle cx="8" cy="8" r="6.6" fill="#CD6799"/>` +
		`<path d="M10.3 5.2c-.4-.6-1.2-.9-2.2-.9-1.3 0-2.2.7-2.2 1.6 0 1.9 4.5 1.4 4.5 3.7 0 1.1-1 1.9-2.4 1.9-1.1 0-2-.4-2.5-1.2" fill="none" stroke="#fff" stroke-width="1.4" stroke-linecap="round"/>`,
	"svg": `<path d="M5.4 1.8h5.2l2.2 5.9L8 14.4 3.2 7.7z" fill="#E9A02F"/>` +
		`<path d="M8 14.2V9.6" stroke="#fff" stroke-width="1" stroke-linecap="round"/><circle cx="8" cy="8.3" r="1.25" fill="#fff"/>`,
	"image": `<rect x="1.5" y="2.5" width="13" height="11" rx="2" fill="#2BA59A"/><circle cx="5.3" cy="6.1" r="1.4" fill="#fff"/>` +
		`<path d="M1.5 11.3 5.4 8.2l2.7 2.2 3.1-3.1 3.3 3.3v1.3a2 2 0 0 1-2 2h-9a2 2 0 0 1-2-2z" fill="#DDF4F1"/>`,
	"sheet": `<rect x="1.5" y="2.5" width="13" height="11" rx="1.8" fill="#2F9E57"/>` +
		`<path d="M1.5 6.2h13M1.5 9.8h13M6 2.5v11" stroke="#fff" stroke-width="1"/>`,
	"database": `<path d="M2.5 4v8.2c0 1.2 2.5 2.3 5.5 2.3s5.5-1 5.5-2.3V4" fill="#D99A33"/>` +
		`<ellipse cx="8" cy="4" rx="5.5" ry="2.3" fill="#EDB75A"/>` +
		`<path d="M2.5 8.1c0 1.2 2.5 2.3 5.5 2.3s5.5-1 5.5-2.3" fill="none" stroke="#fff" stroke-width=".9"/>`,
	"shell": `<rect x="1.5" y="2.5" width="13" height="11" rx="2" fill="#3A4250"/>` +
		`<path d="M4.3 6.1 6.7 8l-2.4 1.9M8.2 10.4h3.4" fill="none" stroke="#7EE787" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>`,
	"powershell": `<rect x="1.5" y="2.5" width="13" height="11" rx="2" fill="#2D6CB5"/>` +
		`<path d="M4.3 6.1 6.7 8l-2.4 1.9M8.2 10.4h3.4" fill="none" stroke="#fff" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>`,

	"python": `<path d="M8 1.5H6.4c-1.1 0-1.8.6-1.8 1.6v1.7h3.8v.6H3.3c-1.1 0-1.8.8-1.8 2.5s.7 2.5 1.8 2.5h1.1V8.9c0-1 .7-1.7 1.7-1.7h3.5c.9 0 1.6-.7 1.6-1.6V3.1c0-1-.8-1.6-1.7-1.6z" fill="#3B77A8"/>` +
		`<path d="M8 1.5H6.4c-1.1 0-1.8.6-1.8 1.6v1.7h3.8v.6H3.3c-1.1 0-1.8.8-1.8 2.5s.7 2.5 1.8 2.5h1.1V8.9c0-1 .7-1.7 1.7-1.7h3.5c.9 0 1.6-.7 1.6-1.6V3.1c0-1-.8-1.6-1.7-1.6z" fill="#F2C53D" transform="rotate(180 8 8)"/>` +
		`<circle cx="6.2" cy="3.1" r=".6" fill="#fff"/><circle cx="9.8" cy="12.9" r=".6" fill="#fff"/>`,
	"rust": `<circle cx="8" cy="8" r="6.2" fill="none" stroke="#DE8A5A" stroke-width="1.8" stroke-dasharray="1.05 .9"/>` +
		`<circle cx="8" cy="8" r="5.1" fill="#DE8A5A"/>` +
		`<path d="M6.3 10.7V5.4h2.1c.9 0 1.5.5 1.5 1.3S9.3 8 8.4 8H6.3M8.6 8l1.4 2.7" fill="none" stroke="#fff" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/>`,
	"c": `<path d="M11.1 5.2a4.2 4.2 0 1 0 0 5.6" fill="none" stroke="#5B8FD6" stroke-width="2" stroke-linecap="round"/>`,
	"cpp": `<path d="M7.6 5.6a3.3 3.3 0 1 0 0 4.8" fill="none" stroke="#3E7FCB" stroke-width="1.9" stroke-linecap="round"/>` +
		`<path d="M9.3 8h2.6M10.6 6.7v2.6M12.4 8H15M13.7 6.7v2.6" stroke="#3E7FCB" stroke-width="1.25" stroke-linecap="round"/>`,
	"csharp": `<path d="M8.4 5.6a3.6 3.6 0 1 0 0 4.8" fill="none" stroke="#A35FB0" stroke-width="1.9" stroke-linecap="round"/>` +
		`<path d="M10.2 6.6h4M10 9.4h4M11.7 4.6l-.6 6.8M13.3 4.6l-.6 6.8" stroke="#A35FB0" stroke-width="1.1" stroke-linecap="round"/>`,
	"h": `<path d="M4.6 3.6v8.8M11.4 3.6v8.8M4.6 8h6.8" stroke="#A074C4" stroke-width="2" stroke-linecap="round"/>`,
	"java": `<path d="M3.4 7.4h7.6v2.8c0 1.9-1.5 3.3-3.4 3.3H6.8c-1.9 0-3.4-1.4-3.4-3.3z" fill="#4F7FA8"/>` +
		`<path d="M11 8.2h.8c.9 0 1.6.7 1.6 1.5s-.7 1.5-1.6 1.5h-1.1" fill="none" stroke="#4F7FA8" stroke-width="1.2"/>` +
		`<path d="M6 2.4c-.7.9.7 1.5 0 2.6M8.6 1.6c-.8 1.1.8 1.8 0 3.2" fill="none" stroke="#E5732F" stroke-width="1.1" stroke-linecap="round"/>`,
	"kotlin": `<defs><linearGradient id="k" x1="2" y1="14" x2="14" y2="2" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="#E44857"/><stop offset=".5" stop-color="#C711E1"/><stop offset="1" stop-color="#7F52FF"/></linearGradient></defs>` +
		`<path d="M2 2h12L8 8l6 6H2z" fill="url(#k)"/>`,
	"swift": `<rect x="1.5" y="1.5" width="13" height="13" rx="3" fill="#F05138"/>` +
		`<path d="M10.9 11.4c-1.6.9-3.8 1-6 .1-1-.4-1.8-1-2.4-1.7.6.4 1.3.7 2 .9 2 .7 3.9.4 5.2-.4-2-1.4-3.7-3.2-5-4.9l-.6-.8c1.6 1.4 4 3.1 4.9 3.6C7 6.3 5.4 4.3 4.6 3.5 6.4 5.3 8.5 6.9 9.3 7.5l.1.1.2-.6c.5-1.6-.1-3.5-1.4-5 2.6 1.6 4.1 4.5 3.5 7l-.1.2c1.2 1.5.9 3 .8 2.8-.6-1.2-1.8-.8-1.5-.6z" fill="#fff"/>`,
	"ruby": `<path d="M4.3 2.5h7.4L14.5 6 8 13.8 1.5 6z" fill="#D4423A"/>` +
		`<path d="M1.5 6h13M6 6l2 7.8L10 6M4.3 2.5 6 6l2-3.5L10 6l1.7-3.5" fill="none" stroke="#F5ABA5" stroke-width=".7" stroke-linejoin="round"/>`,
	"php": `<ellipse cx="8" cy="8" rx="7" ry="4.8" fill="#7A86B8"/>` +
		`<path d="M3.4 10.2V5.9h1.2c.7 0 1.1.4 1.1 1s-.4 1-1.1 1H3.4M6.8 5v5.2M6.8 7.5c.2-.5.6-.8 1.1-.8.6 0 .9.4.9 1v2.5M10.3 10.2V5.9h1.2c.7 0 1.1.4 1.1 1s-.4 1-1.1 1h-1.2" fill="none" stroke="#fff" stroke-width="1" stroke-linecap="round" stroke-linejoin="round"/>`,
	"lua":  `<circle cx="7.3" cy="8.7" r="5.6" fill="#3C64B4"/><circle cx="9.5" cy="6.5" r="1.6" fill="#fff"/><circle cx="13.6" cy="2.4" r="1.4" fill="#3C64B4"/>`,
	"dart": `<path d="M3.7 12.4 1.6 10.3V4.7l3-3.1 7.9 7.9v2.9z" fill="#0B8AD8"/><path d="M4.6 1.6h5.6l4.2 4.2v6.6h-1.9z" fill="#40C4F5"/>`,
	"vue":  `<path d="M1 2.5h3.2L8 9l3.8-6.5H15L8 14.5z" fill="#41B883"/><path d="M4.2 2.5h2.5L8 4.8l1.3-2.3h2.5L8 9z" fill="#4A6681"/>`,
	"svelte": `<path d="M11.6 4.3c-1-1.5-3.1-1.9-4.6-1L4.4 5c-1.4.9-1.8 2.7-1 4.2-.6 1-.5 2.4.3 3.4 1 1.4 3 1.8 4.5.9L10.9 11c1.4-.9 1.8-2.7 1-4.2.5-.9.4-2-.3-2.5z" fill="#FF3E00"/>` +
		`<path d="M9.8 5.7c-.4-.6-1.2-.8-1.8-.4L5.8 6.7c-.5.3-.6 1-.3 1.5.4.6 1.2.8 1.8.4l1.4-.9c.6-.4 1.4-.2 1.8.4.3.5.2 1.2-.3 1.5l-2.2 1.4c-.6.4-1.4.2-1.8-.4" fill="none" stroke="#fff" stroke-width=".9" stroke-linecap="round"/>`,
	"docker": `<path d="M1.5 8.3h11.3c.4-.9 1.1-1.3 2-1.2-.3 3.3-2.6 6.4-7.4 6.4-3.3 0-5.3-1.9-5.9-5.2z" fill="#2496ED"/>` +
		`<path d="M2.9 6.4h1.7v1.5H2.9zM4.9 6.4h1.7v1.5H4.9zM6.9 6.4h1.7v1.5H6.9zM8.9 6.4h1.7v1.5H8.9zM4.9 4.5h1.7V6H4.9zM6.9 4.5h1.7V6H6.9zM8.9 4.5h1.7V6H8.9zM8.9 2.6h1.7v1.5H8.9z" fill="#2496ED"/>`,
	"make": `<path d="M7.4 6.4 2.2 11.6a1.25 1.25 0 0 0 1.8 1.8l5.2-5.2z" fill="#B57B4E"/>` +
		`<path d="M5.7 4.6 8.6 1.7l1.1 1.1.8-.8 3.5 3.5-.8.8.3.3-1.9 1.9-.3-.3-.8.8-1.4-1.4-.9.9-2.6-2.6.9-.9z" fill="#8794A3"/>`,
	"git": `<rect x="2.6" y="2.6" width="10.8" height="10.8" rx="2" transform="rotate(45 8 8)" fill="#F05033"/>` +
		`<path d="M7 5.4v5.2M7 6.6l2.6 2.1" fill="none" stroke="#fff" stroke-width="1.2" stroke-linecap="round"/>` +
		`<g fill="#fff"><circle cx="7" cy="5.3" r="1.15"/><circle cx="7" cy="10.7" r="1.15"/><circle cx="9.8" cy="8.9" r="1.15"/></g>`,
	"lock": `<path d="M5 7.2V5.4a3 3 0 0 1 6 0v1.8" fill="none" stroke="#8C99A8" stroke-width="1.6"/>` +
		`<rect x="3.2" y="7" width="9.6" height="7.2" rx="1.6" fill="#D9A43A"/><circle cx="8" cy="10.4" r="1.05" fill="#fff"/>`,
	"node": `<path d="M8 1.3 13.8 4.65v6.7L8 14.7 2.2 11.35v-6.7z" fill="#5FA04E"/>` +
		`<path d="M6.2 10.4V6l3.6 4.4V6" fill="none" stroke="#fff" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>`,
	"archive": `<rect x="2.5" y="1.5" width="11" height="13" rx="1.6" fill="#B58C4A"/>` +
		`<path d="M7 2.3h1.2M7.8 3.5H9M7 4.7h1.2M7.8 5.9H9M7 7.1h1.2" stroke="#F4E3C1" stroke-width="1"/>` +
		`<rect x="6.6" y="8.3" width="2.8" height="3.4" rx=".7" fill="#F4E3C1"/>`,
	"audio": `<path d="M6 11.6V3.9l7-1.6v7.6" fill="none" stroke="#E5566A" stroke-width="1.5" stroke-linejoin="round"/>` +
		`<ellipse cx="4.2" cy="11.7" rx="2.1" ry="1.7" fill="#E5566A"/><ellipse cx="11.2" cy="10" rx="2.1" ry="1.7" fill="#E5566A"/>`,
	"video": `<rect x="1.5" y="3" width="13" height="10" rx="2" fill="#EE8A3C"/><path d="M6.5 5.7v4.6L10.4 8z" fill="#fff"/>`,
	"font":  `<path d="M3 13.2 8 2.8l5 10.4M4.9 9.6h6.2" fill="none" stroke="#C860B5" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>`,
	"diff": `<path d="M8 2.2v5.4M5.3 4.9h5.4" stroke="#3FA556" stroke-width="1.7" stroke-linecap="round"/>` +
		`<path d="M5.3 11.6h5.4" stroke="#E5534B" stroke-width="1.7" stroke-linecap="round"/>`,
	"graphql": `<g fill="none" stroke="#E535AB" stroke-width=".9"><path d="M8 1.8 13.4 4.9v6.2L8 14.2 2.6 11.1V4.9z"/><path d="M8 1.8 13.4 11.1H2.6z"/></g>` +
		`<g fill="#E535AB"><circle cx="8" cy="1.8" r="1.2"/><circle cx="13.4" cy="4.9" r="1.2"/><circle cx="13.4" cy="11.1" r="1.2"/><circle cx="8" cy="14.2" r="1.2"/><circle cx="2.6" cy="11.1" r="1.2"/><circle cx="2.6" cy="4.9" r="1.2"/></g>`,
	"terraform": `<path d="M6 3.2v3.7l3.2 1.8V5zM9.8 5v3.7L13 6.9V3.2zM2.2 1v3.7l3.2 1.8V2.8zM6 7.6v3.7l3.2 1.8V9.4z" fill="#7B42BC"/>`,
	"binary": `<path d="M6 1.6v2M8 1.6v2M10 1.6v2M6 12.4v2M8 12.4v2M10 12.4v2M1.6 6h2M1.6 8h2M1.6 10h2M12.4 6h2M12.4 8h2M12.4 10h2" stroke="#7D8A99" stroke-width="1" stroke-linecap="round"/>` +
		`<rect x="3.5" y="3.5" width="9" height="9" rx="1.6" fill="#7D8A99"/><rect x="6" y="6" width="4" height="4" rx=".7" fill="#D3DAE2"/>`,
}

// iconByName picks icons by the whole name of a file, in lower case.
var iconByName = map[string]string{
	"license": "license", "licence": "license", "copying": "license", "license.md": "license", "license.txt": "license",
	"licence.md": "license", "licence.txt": "license", "unlicense": "license",
	"makefile": "make", "gnumakefile": "make", "cmakelists.txt": "make", "justfile": "make", "rakefile": "ruby",
	"dockerfile": "docker", "containerfile": "docker", ".dockerignore": "docker",
	"docker-compose.yml": "docker", "docker-compose.yaml": "docker", "compose.yml": "docker", "compose.yaml": "docker",
	".gitignore": "git", ".gitattributes": "git", ".gitmodules": "git", ".gitkeep": "git", ".mailmap": "git",
	"package.json": "node", ".npmrc": "node", ".nvmrc": "node",
	"package-lock.json": "lock", "yarn.lock": "lock", "pnpm-lock.yaml": "lock", "bun.lock": "lock", "bun.lockb": "lock",
	"cargo.lock": "lock", "gemfile.lock": "lock", "poetry.lock": "lock", "composer.lock": "lock", "flake.lock": "lock",
	"cargo.toml": "rust", "go.mod": "go", "go.sum": "go", "go.work": "go", "go.work.sum": "go",
	"gemfile": "ruby", "pyproject.toml": "python", "requirements.txt": "python", "pipfile": "python",
	"tsconfig.json": "ts", "jsconfig.json": "js",
	".editorconfig": "config", ".prettierrc": "config", ".eslintrc": "config", ".clang-format": "config",
	".bashrc": "shell", ".zshrc": "shell", ".profile": "shell", ".bash_profile": "shell", ".zprofile": "shell",
}

// iconByExt picks icons by extension, in lower case, without the dot.
var iconByExt = map[string]string{
	"txt": "text", "text": "text", "log": "text", "rtf": "text", "srt": "text", "vtt": "text", "tex": "text",
	"bib": "text", "rst": "text", "adoc": "text", "org": "text", "strings": "text",
	"md": "markdown", "markdown": "markdown", "mdown": "markdown", "mkd": "markdown", "mkdn": "markdown", "mdx": "markdown",
	"go": "go",
	"js": "js", "mjs": "js", "cjs": "js",
	"ts": "ts", "mts": "ts", "cts": "ts",
	"jsx": "react", "tsx": "react-ts",
	"json": "json", "jsonc": "json", "json5": "json", "jsonl": "json", "ndjson": "json", "webmanifest": "json", "xcstrings": "json",
	"yaml": "yaml", "yml": "yaml",
	"toml": "config", "ini": "config", "cfg": "config", "conf": "config", "properties": "config", "env": "env",
	"xml": "xml", "plist": "xml", "xib": "xml", "storyboard": "xml", "entitlements": "xml", "xsd": "xml", "xsl": "xml",
	"html": "html", "htm": "html", "xhtml": "html",
	"css": "css", "less": "css", "styl": "css", "scss": "sass", "sass": "sass",
	"svg": "svg",
	"png": "image", "jpg": "image", "jpeg": "image", "gif": "image", "webp": "image", "bmp": "image", "ico": "image",
	"icns": "image", "heic": "image", "heif": "image", "tif": "image", "tiff": "image", "avif": "image", "psd": "image",
	"pdf": "pdf",
	"csv": "sheet", "tsv": "sheet", "xls": "sheet", "xlsx": "sheet", "numbers": "sheet",
	"sql": "database", "db": "database", "sqlite": "database", "sqlite3": "database",
	"sh": "shell", "bash": "shell", "zsh": "shell", "fish": "shell", "command": "shell", "ps1": "powershell",
	"psm1": "powershell", "bat": "powershell", "cmd": "powershell",
	"py": "python", "pyi": "python", "pyw": "python", "ipynb": "python",
	"rs": "rust",
	"c":  "c", "m": "c", "h": "h", "hh": "h", "hpp": "h", "hxx": "h",
	"cc": "cpp", "cpp": "cpp", "cxx": "cpp", "c++": "cpp", "mm": "cpp", "cs": "csharp",
	"java": "java", "class": "java", "jar": "java", "gradle": "java",
	"kt": "kotlin", "kts": "kotlin",
	"swift": "swift",
	"rb":    "ruby", "gemspec": "ruby", "erb": "ruby",
	"php":  "php",
	"lua":  "lua",
	"dart": "dart",
	"vue":  "vue", "svelte": "svelte",
	"dockerfile": "docker",
	"mk":         "make", "cmake": "make",
	"lock": "lock",
	"zip":  "archive", "tar": "archive", "gz": "archive", "tgz": "archive", "bz2": "archive", "xz": "archive",
	"7z": "archive", "rar": "archive", "zst": "archive", "dmg": "archive", "pkg": "archive",
	"mp3": "audio", "wav": "audio", "flac": "audio", "aac": "audio", "m4a": "audio", "ogg": "audio", "opus": "audio",
	"aif": "audio", "aiff": "audio", "mid": "audio",
	"mp4": "video", "mov": "video", "mkv": "video", "webm": "video", "avi": "video", "m4v": "video",
	"ttf": "font", "otf": "font", "woff": "font", "woff2": "font", "ttc": "font",
	"diff": "diff", "patch": "diff",
	"graphql": "graphql", "gql": "graphql",
	"tf": "terraform", "tfvars": "terraform", "hcl": "terraform",
	"exe": "binary", "bin": "binary", "o": "binary", "a": "binary", "so": "binary", "dylib": "binary",
	"wasm": "binary", "dll": "binary",
	"proto": "code", "zig": "code", "nim": "code", "ex": "code", "exs": "code", "erl": "code", "hs": "code",
	"ml": "code", "fs": "code", "clj": "code", "scala": "code", "pl": "code", "pm": "code", "r": "code",
	"astro": "code", "groovy": "code", "vim": "code", "el": "code", "jl": "code",
}

// iconName returns the icon of a file or a folder named name.
func iconName(name string, dir, open bool) string {
	if dir {
		n := "folder"
		if strings.HasPrefix(name, ".") {
			n = "folder-dot"
		}
		if open {
			n += "-open"
		}
		return n
	}
	lower := strings.ToLower(name)
	if n, ok := iconByName[lower]; ok {
		return n
	}
	switch {
	case strings.HasPrefix(lower, "readme"):
		return "readme"
	case strings.HasPrefix(lower, "license") || strings.HasPrefix(lower, "licence"):
		return "license"
	case lower == ".env" || strings.HasPrefix(lower, ".env."):
		return "env"
	case strings.HasPrefix(lower, "dockerfile") || strings.HasSuffix(lower, ".dockerfile"):
		return "docker"
	case strings.HasPrefix(lower, "tsconfig.") && strings.HasSuffix(lower, ".json"):
		return "ts"
	case strings.HasSuffix(lower, ".d.ts"):
		return "ts"
	}
	if n, ok := iconByExt[strings.TrimPrefix(filepath.Ext(lower), ".")]; ok {
		return n
	}
	if strings.HasPrefix(lower, ".") && strings.HasSuffix(lower, "rc") {
		return "config" // .prettierrc, .vimrc
	}
	return "file"
}

var (
	iconsMu sync.Mutex
	icons   = map[string]*ui.SVG{}
)

// fileIcon returns the parsed icon of a file or a folder named name.
// Main thread.
func fileIcon(name string, dir, open bool) *ui.SVG {
	return iconNamed(iconName(name, dir, open))
}

func iconNamed(n string) *ui.SVG {
	iconsMu.Lock()
	defer iconsMu.Unlock()
	if s, ok := icons[n]; ok {
		return s
	}
	s := ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16">` + fileIcons[n] + `</svg>`))
	icons[n] = s
	return s
}

// folderIcon is a folder of two colors, its back and its front; open, the
// front leans forward.
func folderIcon(back, front string, open bool) string {
	if open {
		return `<path d="M1.5 4.3c0-.7.6-1.3 1.3-1.3h3.1c.4 0 .7.1 1 .4L8.1 4.6h4.1c.7 0 1.3.6 1.3 1.3V7H4.6c-.6 0-1.1.4-1.3.9L1.5 12.6z" fill="` + back + `"/>` +
			`<path d="M3.4 7.9c.2-.5.7-.9 1.3-.9h9.7c.7 0 1.2.7.9 1.3l-1.9 4.8c-.2.5-.7.9-1.2.9H1.9c-.5 0-.8-.5-.6-1z" fill="` + front + `"/>`
	}
	return `<path d="M1.5 4.3c0-.7.6-1.3 1.3-1.3h3.1c.4 0 .7.1 1 .4L8.1 4.6h5.1c.7 0 1.3.6 1.3 1.3v6.8c0 .7-.6 1.3-1.3 1.3H2.8c-.7 0-1.3-.6-1.3-1.3z" fill="` + back + `"/>` +
		`<path d="M1.5 6.6c0-.7.6-1.3 1.3-1.3h10.4c.7 0 1.3.6 1.3 1.3v6.1c0 .7-.6 1.3-1.3 1.3H2.8c-.7 0-1.3-.6-1.3-1.3z" fill="` + front + `"/>`
}

// page is a sheet with its corner folded, and what goes on it.
func page(color, fold, inner string) string {
	return `<path d="M4.2 1.5h5l3.3 3.3v8.7c0 .6-.4 1-1 1H4.2c-.6 0-1-.4-1-1v-11c0-.6.4-1 1-1z" fill="` + color + `"/>` +
		`<path d="M9.2 1.5v2.8c0 .3.2.5.5.5h2.8z" fill="` + fold + `"/>` + inner
}

// atom is React's: three orbits around a nucleus.
func atom(color string) string {
	e := `<ellipse cx="8" cy="8" rx="6.7" ry="2.6"`
	return `<g fill="none" stroke="` + color + `" stroke-width="1">` + e + `/>` + e + ` transform="rotate(60 8 8)"/>` + e + ` transform="rotate(120 8 8)"/></g>` +
		`<circle cx="8" cy="8" r="1.4" fill="` + color + `"/>`
}

// gear is a cog of 8 teeth with a hole.
func gear(color string) string {
	const teeth, outer, inner, hole = 8, 6.6, 4.9, 2.1
	var b strings.Builder
	step := 2 * math.Pi / teeth
	pt := func(cmd string, r, a float64) {
		fmt.Fprintf(&b, "%s%.2f %.2f", cmd, 8+r*math.Cos(a), 8+r*math.Sin(a))
	}
	for i := range teeth {
		a := float64(i) * step
		cmd := "L"
		if i == 0 {
			cmd = "M"
		}
		pt(cmd, inner, a-step*0.36)
		pt("L", outer, a-step*0.2)
		pt("L", outer, a+step*0.2)
		pt("L", inner, a+step*0.36)
	}
	b.WriteString("Z")
	fmt.Fprintf(&b, "M%.2f 8a%.2f %.2f 0 1 0 %.2f 0a%.2f %.2f 0 1 0 %.2f 0Z", 8+hole, hole, hole, -2*hole, hole, hole, 2*hole)
	return `<path d="` + b.String() + `" fill="` + color + `" fill-rule="evenodd"/>`
}
