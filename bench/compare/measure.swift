// measure: opens a file in an app, as a double click in Finder does, and
// times its window coming on screen and its content settling.
//
//   measure --app X.app --file F [--warm] [--arg A]... [--exec CMD] [--timeout 30] [--settle 1500]
//
// --exec runs CMD F instead, as a command such as qe does.
//
// Prints one line of JSON:
//   windowMs  a window of the app titled with the file's name is on screen
//   visualMs  the last time that window's pixels changed, before staying
//             the same for settle ms (a blinking caret and fading scroll
//             bars change too few pixels to count): text, highlighting and
//             rendered Markdown included
//   frontMs   that window is the frontmost one on screen
//   memoryMB  the physical footprint of every process of the app, 2 s later
import AppKit
import CoreGraphics
import Darwin
import ImageIO

setvbuf(stdout, nil, _IOLBF, 0)
var appPath = "", filePath = "", warm = false, extraArgs: [String] = [], execPath = ""
var timeout = 30.0, settle = 1.5
var it = CommandLine.arguments.dropFirst().makeIterator()
while let a = it.next() {
    switch a {
    case "--app": appPath = it.next()!
    case "--file": filePath = it.next()!
    case "--warm": warm = true
    case "--arg": extraArgs.append(it.next()!)
    case "--exec": execPath = it.next()!
    case "--timeout": timeout = Double(it.next()!)!
    case "--settle": settle = Double(it.next()!)! / 1000
    default: fatalError("unknown argument \(a)")
    }
}
let appURL = URL(fileURLWithPath: appPath), fileURL = URL(fileURLWithPath: filePath)
let bundleID = Bundle(url: appURL)!.bundleIdentifier!
let name = fileURL.lastPathComponent

func emit(_ d: [String: Any]) {
    var d = d
    d["app"] = appURL.deletingPathExtension().lastPathComponent
    d["file"] = name
    d["mode"] = warm ? "warm" : "cold"
    let data = try! JSONSerialization.data(withJSONObject: d, options: [.sortedKeys])
    print(String(data: data, encoding: .utf8)!)
}

func now() -> Double { Double(DispatchTime.now().uptimeNanoseconds) / 1e9 }

func pids() -> [pid_t] { NSRunningApplication.runningApplications(withBundleIdentifier: bundleID).map(\.processIdentifier) }

if !warm && !pids().isEmpty {
    emit(["error": "already running"]); exit(1)
}

// The window of the app whose title names the file.
func findWindow(_ owners: Set<pid_t>) -> CGWindowID? {
    guard let list = CGWindowListCopyWindowInfo([.optionOnScreenOnly, .excludeDesktopElements], kCGNullWindowID) as? [[String: Any]] else { return nil }
    for w in list {
        guard let pid = w[kCGWindowOwnerPID as String] as? pid_t, owners.contains(pid),
              (w[kCGWindowLayer as String] as? Int) == 0,
              let title = w[kCGWindowName as String] as? String, title.contains(name),
              let id = w[kCGWindowNumber as String] as? CGWindowID else { continue }
        return id
    }
    return nil
}

// isFront reports whether id is the frontmost window of the screen.
func isFront(_ id: CGWindowID) -> Bool {
    guard let list = CGWindowListCopyWindowInfo([.optionOnScreenOnly, .excludeDesktopElements], kCGNullWindowID) as? [[String: Any]] else { return false }
    let first = list.first { ($0[kCGWindowLayer as String] as? Int) == 0 }
    return (first?[kCGWindowNumber as String] as? CGWindowID) == id
}

// CGWindowListCreateImage is unavailable to Swift since macOS 15, but
// still there: it captures a window in a few milliseconds.
typealias CreateImage = @convention(c) (CGRect, UInt32, UInt32, UInt32) -> Unmanaged<CGImage>?
let cg = dlopen("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics", RTLD_NOW)
let createImage = unsafeBitCast(dlsym(cg, "CGWindowListCreateImage")!, to: CreateImage.self)

// A capture, scaled to a thumbnail of a fixed size, so that the window's
// opening animation, which scales it by a fraction of a percent, leaves it
// nearly the same, while text appearing or being colored changes it.
let thumbW = 320, thumbH = 224
struct Shot { var px: [UInt8] = [] }
func capture(_ id: CGWindowID) -> Shot? {
    // including the window alone, ignoring its frame, at 1x
    guard let img = createImage(.null, 1 << 3, id, 1 << 0 | 1 << 4)?.takeRetainedValue() else { return nil }
    var s = Shot(px: [UInt8](repeating: 0, count: thumbW * thumbH * 4))
    let ok = s.px.withUnsafeMutableBytes { buf -> Bool in
        guard let ctx = CGContext(data: buf.baseAddress, width: thumbW, height: thumbH, bitsPerComponent: 8, bytesPerRow: thumbW * 4,
                                  space: CGColorSpaceCreateDeviceRGB(), bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue) else { return false }
        ctx.interpolationQuality = .medium
        ctx.draw(img, in: CGRect(x: 0, y: 0, width: thumbW, height: thumbH))
        return true
    }
    return ok ? s : nil
}

let debug = ProcessInfo.processInfo.environment["MEASURE_DEBUG"] != nil
// changed reports whether more than 1% of the thumbnail's pixels changed.
func changed(_ a: Shot, _ b: Shot) -> Bool {
    var n = 0
    let limit = thumbW * thumbH / 100
    a.px.withUnsafeBufferPointer { pa in
        b.px.withUnsafeBufferPointer { pb in
            var i = 0
            while i < pa.count {
                if abs(Int(pa[i]) - Int(pb[i])) > 24 || abs(Int(pa[i + 1]) - Int(pb[i + 1])) > 24 || abs(Int(pa[i + 2]) - Int(pb[i + 2])) > 24 {
                    n += 1
                }
                i += 4
            }
        }
    }
    if debug && n > 0 { print(String(format: "  %.0f ms: %d px changed%@", (now() - t0) * 1000, n, n > limit ? " (counts)" : "")) }
    return n > limit
}

// footprint sums the physical footprint of the processes running code of the app's bundle.
func footprint() -> (Double, Int) {
    let n = proc_listallpids(nil, 0)
    var all = [pid_t](repeating: 0, count: Int(n) * 2)
    let got = proc_listallpids(&all, Int32(all.count * MemoryLayout<pid_t>.size))
    var total = 0.0, count = 0
    var path = [CChar](repeating: 0, count: 4096)
    for pid in all.prefix(Int(got)) where pid > 0 {
        guard proc_pidpath(pid, &path, UInt32(path.count)) > 0 else { continue }
        let p = String(cString: path)
        guard p.hasPrefix(appURL.path + "/") || p.hasPrefix(appURL.resolvingSymlinksInPath().path + "/") else { continue }
        var info = rusage_info_v4()
        let r = withUnsafeMutablePointer(to: &info) { p in
            p.withMemoryRebound(to: rusage_info_t?.self, capacity: 1) { proc_pid_rusage(pid, RUSAGE_INFO_V4, $0) }
        }
        if r == 0 { total += Double(info.ri_phys_footprint); count += 1 }
    }
    return (total / 1e6, count)
}

var t0 = 0.0
let cfg = NSWorkspace.OpenConfiguration()
cfg.activates = true
cfg.arguments = extraArgs
t0 = now()
if debug { print(String(format: "t0 unix %.1f", Date().timeIntervalSince1970 * 1000)) }
if execPath != "" {
    let p = Process()
    p.executableURL = URL(fileURLWithPath: execPath)
    p.arguments = [filePath]
    do { try p.run() } catch { emit(["error": "exec: \(error)"]); exit(1) }
} else {
    NSWorkspace.shared.open([fileURL], withApplicationAt: appURL, configuration: cfg) { _, err in
        if let err { emit(["error": "open: \(err.localizedDescription)"]); exit(1) }
    }
}

var windowAt = 0.0, lastChange = 0.0, frontAt = 0.0
var window: CGWindowID = 0
var last: Shot?
while true {
    let t = now()
    if t - t0 > timeout {
        emit(["error": window == 0 ? "no window within \(Int(timeout)) s" : (frontAt == 0 ? "never in front" : "never settled"), "windowMs": windowAt > 0 ? (windowAt - t0) * 1000 : -1])
        exit(1)
    }
    if window == 0 {
        if let id = findWindow(Set(pids())) {
            window = id
            windowAt = now()
            // Two copies of an app share its bundle identifier, and the
            // system may start the other: measure only the one asked for.
            let want = appURL.resolvingSymlinksInPath().path
            let ran = NSRunningApplication.runningApplications(withBundleIdentifier: bundleID).compactMap { $0.bundleURL?.resolvingSymlinksInPath().path }
            if !ran.contains(want) {
                emit(["error": "the system ran \(ran.joined(separator: ", ")), not \(want)"]); exit(1)
            }
            if debug { print(String(format: "window at %.0f ms", (windowAt - t0) * 1000)) }
        } else {
            RunLoop.current.run(until: Date(timeIntervalSinceNow: 0.003))
            continue
        }
    }
    if frontAt == 0 && isFront(window) { frontAt = now() }
    let c0 = now()
    let maybeShot = capture(window)
    if debug { print(String(format: "capture at %.0f took %.1f ms: %@", (c0 - t0) * 1000, (now() - c0) * 1000, maybeShot == nil ? "nil" : "ok")) }
    if let shot = maybeShot {
        // The capture shows the window as it was when asked: the first of a
        // new window takes the window server tens of milliseconds.
        let at = c0
        if last == nil || changed(last!, shot) {
            lastChange = at
            last = shot
            if let dir = ProcessInfo.processInfo.environment["MEASURE_SHOTS"], let img = createImage(.null, 1 << 3, window, 1 << 0 | 1 << 4)?.takeRetainedValue() {
                let url = URL(fileURLWithPath: dir).appendingPathComponent(String(format: "%04.0f.png", (at - t0) * 1000))
                if let dst = CGImageDestinationCreateWithURL(url as CFURL, "public.png" as CFString, 1, nil) { CGImageDestinationAddImage(dst, img, nil); CGImageDestinationFinalize(dst) }
            }
        }
    }
    if now() - lastChange > settle && frontAt > 0 { break }
    RunLoop.current.run(until: Date(timeIntervalSinceNow: 0.005))
}
Thread.sleep(forTimeInterval: 2)
let (mem, procs) = footprint()
emit(["windowMs": ((windowAt - t0) * 1000).rounded(), "visualMs": ((lastChange - t0) * 1000).rounded(), "frontMs": ((frontAt - t0) * 1000).rounded(), "memoryMB": mem.rounded(), "procs": procs])
