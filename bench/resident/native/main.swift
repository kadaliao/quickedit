// resident-native: AppKit app that stays running and opens each file it is
// handed in a new window with a TextKit 2 NSTextView (plain text).
import AppKit

let logPath = ProcessInfo.processInfo.environment["QE_LOG"]
func logLine(_ s: String) {
    guard let p = logPath, let h = FileHandle(forWritingAtPath: p) ?? { FileManager.default.createFile(atPath: p, contents: nil); return FileHandle(forWritingAtPath: p) }() else { return }
    h.seekToEndOfFile(); h.write((s + "\n").data(using: .utf8)!); h.closeFile()
}

final class Delegate: NSObject, NSApplicationDelegate {
    var windows: [NSWindow] = []
    func applicationDidFinishLaunching(_ n: Notification) {
        let menu = NSMenu(); let item = NSMenuItem(); menu.addItem(item)
        let app = NSMenu(); app.addItem(withTitle: "Quit", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        item.submenu = app; NSApp.mainMenu = menu
    }
    func applicationShouldTerminateAfterLastWindowClosed(_ s: NSApplication) -> Bool { false }
    func application(_ sender: NSApplication, openFiles filenames: [String]) {
        for path in filenames { open(path) }
    }
    func open(_ path: String) {
        let t0 = CFAbsoluteTimeGetCurrent()
        let text = (try? String(contentsOfFile: path, encoding: .utf8)) ?? ""
        let w = NSWindow(contentRect: NSRect(x: 200, y: 200, width: 1000, height: 700),
                         styleMask: [.titled, .closable, .miniaturizable, .resizable], backing: .buffered, defer: false)
        w.isReleasedWhenClosed = false
        w.title = (path as NSString).lastPathComponent
        let scroll = NSTextView.scrollableTextView()
        let tv = scroll.documentView as! NSTextView
        tv.font = .monospacedSystemFont(ofSize: 13, weight: .regular)
        tv.isRichText = false
        tv.string = text
        scroll.frame = w.contentView!.bounds
        scroll.autoresizingMask = [.width, .height]
        w.contentView!.addSubview(scroll)
        windows.append(w)
        w.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
        w.displayIfNeeded()
        CATransaction.setCompletionBlock { logLine(String(format: "%@ %.1f", w.title, (CFAbsoluteTimeGetCurrent() - t0) * 1000)) }
        CATransaction.flush()
    }
}
let app = NSApplication.shared
let d = Delegate(); app.delegate = d
app.setActivationPolicy(.regular)
app.run()
