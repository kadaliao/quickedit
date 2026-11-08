// bench-native: the floor. A plain AppKit window with a TextKit 2 NSTextView
// showing the file as plain text (no highlighting, no Markdown rendering).
import AppKit

setvbuf(stdout, nil, _IOLBF, 0)
let path = CommandLine.arguments.count > 1 ? CommandLine.arguments[1] : ""
let text = (try? String(contentsOfFile: path, encoding: .utf8)) ?? ""

final class Delegate: NSObject, NSApplicationDelegate {
    var window: NSWindow!
    func applicationDidFinishLaunching(_ n: Notification) {
        let menu = NSMenu(); let item = NSMenuItem(); menu.addItem(item)
        let app = NSMenu(); app.addItem(withTitle: "Quit", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        item.submenu = app; NSApp.mainMenu = menu

        window = NSWindow(contentRect: NSRect(x: 200, y: 200, width: 1000, height: 700),
                          styleMask: [.titled, .closable, .miniaturizable, .resizable], backing: .buffered, defer: false)
        window.title = (path as NSString).lastPathComponent
        let scroll = NSTextView.scrollableTextView()
        let tv = scroll.documentView as! NSTextView
        tv.font = .monospacedSystemFont(ofSize: 13, weight: .regular)
        tv.isRichText = false
        tv.string = text
        scroll.frame = window.contentView!.bounds
        scroll.autoresizingMask = [.width, .height]
        window.contentView!.addSubview(scroll)
        window.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
        window.displayIfNeeded()
        CATransaction.setCompletionBlock { print("ready"); print("highlighted") }
        CATransaction.flush()
    }
}
let app = NSApplication.shared
let d = Delegate(); app.delegate = d
app.setActivationPolicy(.regular)
app.run()
