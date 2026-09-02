import AppKit
import Foundation

func parseRequest() throws -> FideliusRequest {
    let args = Array(CommandLine.arguments.dropFirst())
    var service = ""
    var accounts: [String] = []
    var index = 0

    while index < args.count {
        let arg = args[index]
        if arg == "--service" {
            guard index + 1 < args.count else {
                throw NSError(domain: "Fidelius", code: 64, userInfo: [NSLocalizedDescriptionKey: "--service requires a value"])
            }
            index += 1
            service = args[index]
        } else {
            accounts.append(arg)
        }
        index += 1
    }

    guard !service.isEmpty, !accounts.isEmpty else {
        throw NSError(domain: "Fidelius", code: 64, userInfo: [NSLocalizedDescriptionKey: "missing service or API key names"])
    }
    return FideliusRequest(service: service, accounts: accounts)
}

do {
    let request = try parseRequest()
    let app = NSApplication.shared
    let delegate = AppDelegate(request: request)
    app.setActivationPolicy(.regular)
    app.delegate = delegate
    app.run()
} catch {
    fputs("\(error.localizedDescription)\n", stderr)
    exit(64)
}
