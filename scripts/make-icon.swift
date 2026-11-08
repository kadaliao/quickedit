// Draws resources/icon.png, QuickEdit's 1024×1024 app icon:
// swift scripts/make-icon.swift resources/icon.png
import AppKit

let size = 1024.0
let out = CommandLine.arguments.count > 1 ? CommandLine.arguments[1] : "icon.png"
let rep = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: Int(size), pixelsHigh: Int(size), bitsPerSample: 8,
                           samplesPerPixel: 4, hasAlpha: true, isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)!
NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: rep)
let cg = NSGraphicsContext.current!.cgContext
let rgb = CGColorSpaceCreateDeviceRGB()
func color(_ hex: UInt32, _ a: CGFloat = 1) -> CGColor {
    CGColor(red: CGFloat(hex >> 16 & 0xff) / 255, green: CGFloat(hex >> 8 & 0xff) / 255, blue: CGFloat(hex & 0xff) / 255, alpha: a)
}

// The tile, on Apple's icon grid: 824 points with a little shadow.
let tile = CGRect(x: 100, y: 100, width: 824, height: 824)
let tilePath = CGPath(roundedRect: tile, cornerWidth: 186, cornerHeight: 186, transform: nil)
cg.saveGState()
cg.setShadow(offset: CGSize(width: 0, height: -12), blur: 28, color: color(0x000000, 0.28))
cg.addPath(tilePath); cg.setFillColor(color(0x3355ee)); cg.fillPath()
cg.restoreGState()
cg.saveGState()
cg.addPath(tilePath); cg.clip()
let bg = CGGradient(colorsSpace: rgb, colors: [color(0x5b8cff), color(0x2b3fd4)] as CFArray, locations: [0, 1])!
cg.drawLinearGradient(bg, start: CGPoint(x: 0, y: tile.maxY), end: CGPoint(x: 0, y: tile.minY), options: [])
cg.restoreGState()

// The page, with a folded corner.
let page = CGRect(x: 292, y: 222, width: 400, height: 520)
let fold = 96.0
let pagePath = CGMutablePath()
pagePath.move(to: CGPoint(x: page.minX + 36, y: page.minY))
pagePath.addLine(to: CGPoint(x: page.maxX - 36, y: page.minY))
pagePath.addQuadCurve(to: CGPoint(x: page.maxX, y: page.minY + 36), control: CGPoint(x: page.maxX, y: page.minY))
pagePath.addLine(to: CGPoint(x: page.maxX, y: page.maxY - fold))
pagePath.addLine(to: CGPoint(x: page.maxX - fold, y: page.maxY))
pagePath.addLine(to: CGPoint(x: page.minX + 36, y: page.maxY))
pagePath.addQuadCurve(to: CGPoint(x: page.minX, y: page.maxY - 36), control: CGPoint(x: page.minX, y: page.maxY))
pagePath.addLine(to: CGPoint(x: page.minX, y: page.minY + 36))
pagePath.addQuadCurve(to: CGPoint(x: page.minX + 36, y: page.minY), control: CGPoint(x: page.minX, y: page.minY))
cg.saveGState()
cg.setShadow(offset: CGSize(width: 0, height: -10), blur: 24, color: color(0x0a1a6b, 0.35))
cg.addPath(pagePath); cg.setFillColor(color(0xffffff)); cg.fillPath()
cg.restoreGState()
let foldPath = CGMutablePath()
foldPath.move(to: CGPoint(x: page.maxX, y: page.maxY - fold))
foldPath.addLine(to: CGPoint(x: page.maxX - fold + 18, y: page.maxY - fold))
foldPath.addQuadCurve(to: CGPoint(x: page.maxX - fold, y: page.maxY - fold + 18), control: CGPoint(x: page.maxX - fold, y: page.maxY - fold))
foldPath.addLine(to: CGPoint(x: page.maxX - fold, y: page.maxY))
foldPath.closeSubpath()
cg.addPath(foldPath); cg.setFillColor(color(0xd5ddf5)); cg.fillPath()

// Lines of code on the page.
let lines: [(CGFloat, CGFloat, UInt32)] = [(0, 170, 0xc678dd), (40, 210, 0x9aa6c4), (40, 150, 0x4f7cff), (0, 120, 0x9aa6c4),
                                           (0, 200, 0xc678dd), (40, 230, 0x9aa6c4), (40, 110, 0x4f7cff)]
for (i, l) in lines.enumerated() {
    let y = page.maxY - 150 - CGFloat(i) * 46
    let r = CGRect(x: page.minX + 52 + l.0, y: y, width: l.1, height: 20)
    cg.addPath(CGPath(roundedRect: r, cornerWidth: 10, cornerHeight: 10, transform: nil))
    cg.setFillColor(color(l.2)); cg.fillPath()
}

// The bolt: quick.
let bolt = CGMutablePath()
let o = CGPoint(x: 600, y: 180)
for (i, p) in [(70, 300), (-10, 300), (40, 470), (-120, 250), (-40, 250), (-90, 80)].enumerated() {
    let pt = CGPoint(x: o.x + CGFloat(p.0), y: o.y + CGFloat(p.1) - 80)
    if i == 0 { bolt.move(to: pt) } else { bolt.addLine(to: pt) }
}
bolt.closeSubpath()
cg.saveGState()
cg.setShadow(offset: CGSize(width: 0, height: -8), blur: 18, color: color(0x7a2e00, 0.4))
cg.addPath(bolt); cg.clip()
let gold = CGGradient(colorsSpace: rgb, colors: [color(0xffd84a), color(0xff8a1f)] as CFArray, locations: [0, 1])!
cg.drawLinearGradient(gold, start: CGPoint(x: 0, y: 600), end: CGPoint(x: 0, y: 100), options: [])
cg.restoreGState()
cg.addPath(bolt); cg.setStrokeColor(color(0xffffff, 0.9)); cg.setLineWidth(10); cg.setLineJoin(.round); cg.strokePath()

NSGraphicsContext.current = nil
try! rep.representation(using: .png, properties: [:])!.write(to: URL(fileURLWithPath: out))
print("wrote \(out)")
