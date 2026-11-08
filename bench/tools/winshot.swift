// winshot <pid> <out.png>: captures the first on-screen window of a process.
import Foundation
import CoreGraphics

let args = CommandLine.arguments
guard args.count == 3, let pid = Int32(args[1]) else {
    FileHandle.standardError.write("usage: winshot <pid> <out.png>\n".data(using: .utf8)!)
    exit(2)
}
let list = CGWindowListCopyWindowInfo([.optionOnScreenOnly], kCGNullWindowID) as! [[String: Any]]
guard let w = list.first(where: { ($0[kCGWindowOwnerPID as String] as? Int32) == pid && ($0[kCGWindowLayer as String] as? Int) == 0 }),
      let id = w[kCGWindowNumber as String] as? Int else {
    FileHandle.standardError.write("no on-screen window for pid \(pid)\n".data(using: .utf8)!)
    exit(1)
}
let p = Process()
p.executableURL = URL(fileURLWithPath: "/usr/sbin/screencapture")
p.arguments = ["-x", "-o", "-l\(id)", args[2]]
try p.run(); p.waitUntilExit()
exit(p.terminationStatus)
