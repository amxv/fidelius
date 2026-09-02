import AppKit
import Foundation
import QuartzCore

struct FideliusRequest {
    let message: String?
    let accounts: [String]
    let autoDeleteLabel: String
    let secretFD: Int32
}

private struct PromptResponse: Codable {
    let cancelled: Bool
    let values: [String: String]
}

private final class FideliusSecureTextField: NSSecureTextField {
    override init(frame frameRect: NSRect) {
        super.init(frame: frameRect)
        configure()
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        configure()
    }

    private func configure() {
        focusRingType = .none
        wantsLayer = true
        layer?.cornerRadius = 6
        layer?.masksToBounds = false
    }

    func setFocused(_ focused: Bool) {
        guard let layer else { return }
        CATransaction.begin()
        CATransaction.setDisableActions(true)
        layer.borderWidth = focused ? 2 : 0
        layer.borderColor = focused ? NSColor.controlAccentColor.cgColor : NSColor.clear.cgColor
        CATransaction.commit()
    }
}

private final class AdaptiveTintView: NSView {
    override init(frame frameRect: NSRect) {
        super.init(frame: frameRect)
        wantsLayer = true
        updateTint()
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        wantsLayer = true
        updateTint()
    }

    override func viewDidChangeEffectiveAppearance() {
        super.viewDidChangeEffectiveAppearance()
        updateTint()
    }

    private func updateTint() {
        guard let layer else { return }
        let appearance = effectiveAppearance.bestMatch(from: [.darkAqua, .aqua])
        let color: NSColor
        if appearance == .darkAqua {
            color = NSColor.black.withAlphaComponent(0.34)
        } else {
            color = NSColor.white.withAlphaComponent(0.38)
        }
        layer.backgroundColor = color.cgColor
    }
}

final class AppDelegate: NSObject, NSApplicationDelegate, NSWindowDelegate, NSTextFieldDelegate {
    private let request: FideliusRequest
    private var window: NSWindow?
    private var fields: [String: FideliusSecureTextField] = [:]
    private var saveButton: NSButton?
    private var didFinish = false

    init(request: FideliusRequest) {
        self.request = request
        super.init()
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        buildMenus()
        showWindow()
    }

    func applicationShouldTerminate(_ sender: NSApplication) -> NSApplication.TerminateReply {
        if !didFinish {
            emit(PromptResponse(cancelled: true, values: [:]))
            didFinish = true
        }
        return .terminateNow
    }

    func windowWillClose(_ notification: Notification) {
        cancel(nil)
    }

    func controlTextDidBeginEditing(_ obj: Notification) {
        (obj.object as? FideliusSecureTextField)?.setFocused(true)
    }

    func controlTextDidEndEditing(_ obj: Notification) {
        (obj.object as? FideliusSecureTextField)?.setFocused(false)
    }

    func controlTextDidChange(_ obj: Notification) {
        updateSaveButton()
    }

    @objc private func save(_ sender: Any?) {
        let missing = request.accounts.first { account in
            fields[account]?.stringValue.isEmpty ?? true
        }
        if let missing, let field = fields[missing] {
            NSSound.beep()
            window?.makeFirstResponder(field)
            return
        }

        var values: [String: String] = [:]
        for account in request.accounts {
            guard let value = fields[account]?.stringValue else { continue }
            values[account] = value
        }
        finish(PromptResponse(cancelled: false, values: values))
    }

    @objc private func cancel(_ sender: Any?) {
        guard !didFinish else { return }
        finish(PromptResponse(cancelled: true, values: [:]))
    }

    private func finish(_ response: PromptResponse) {
        guard !didFinish else { return }
        didFinish = true
        emit(response)
        NSApp.terminate(nil)
    }

    private func emit(_ response: PromptResponse) {
        guard let data = try? JSONEncoder().encode(response) else {
            fputs("Fidelius could not encode the prompt response.\n", stderr)
            return
        }
        let handle = FileHandle(fileDescriptor: request.secretFD, closeOnDealloc: false)
        handle.write(data)
        handle.write(Data("\n".utf8))
    }

    private func updateSaveButton() {
        saveButton?.isEnabled = request.accounts.allSatisfy { account in
            !(fields[account]?.stringValue.isEmpty ?? true)
        }
    }

    private func showWindow() {
        let contentWidth: CGFloat = 376
        let frame = NSRect(x: 0, y: 0, width: 420, height: 250)
        let window = NSWindow(
            contentRect: frame,
            styleMask: [.titled, .closable, .miniaturizable, .fullSizeContentView],
            backing: .buffered,
            defer: false
        )
        window.title = "Fidelius"
        window.titlebarAppearsTransparent = true
        window.titleVisibility = .visible
        window.backgroundColor = .clear
        window.isOpaque = false
        window.level = .normal
        window.animationBehavior = .none
        window.isMovableByWindowBackground = true
        window.delegate = self
        window.isReleasedWhenClosed = false
        self.window = window

        let root = NSStackView()
        root.orientation = .vertical
        root.alignment = .leading
        root.spacing = 11
        root.translatesAutoresizingMaskIntoConstraints = false

        let header = makeHeader(width: contentWidth)
        root.addArrangedSubview(header)
        root.setCustomSpacing(15, after: header)

        let fieldStack = makeFieldStack(width: contentWidth)
        if request.accounts.count <= 5 {
            root.addArrangedSubview(fieldStack)
        } else {
            let scrollView = NSScrollView()
            scrollView.drawsBackground = false
            scrollView.hasVerticalScroller = true
            scrollView.borderType = .noBorder
            scrollView.translatesAutoresizingMaskIntoConstraints = false

            let document = NSView()
            document.translatesAutoresizingMaskIntoConstraints = false
            document.addSubview(fieldStack)
            NSLayoutConstraint.activate([
                fieldStack.leadingAnchor.constraint(equalTo: document.leadingAnchor, constant: 2),
                fieldStack.trailingAnchor.constraint(equalTo: document.trailingAnchor, constant: -2),
                fieldStack.topAnchor.constraint(equalTo: document.topAnchor, constant: 2),
                fieldStack.bottomAnchor.constraint(equalTo: document.bottomAnchor, constant: -2),
                document.widthAnchor.constraint(equalToConstant: contentWidth),
            ])
            scrollView.documentView = document
            scrollView.widthAnchor.constraint(equalToConstant: contentWidth).isActive = true
            scrollView.heightAnchor.constraint(equalToConstant: 290).isActive = true
            root.addArrangedSubview(scrollView)
        }

        root.setCustomSpacing(12, after: fieldStack)
        root.addArrangedSubview(makeAutoDeleteRow())
        root.setCustomSpacing(15, after: root.arrangedSubviews.last!)
        root.addArrangedSubview(makeButtons(width: contentWidth))

        let content = NSVisualEffectView()
        content.material = .underWindowBackground
        content.blendingMode = .behindWindow
        content.state = .active
        content.translatesAutoresizingMaskIntoConstraints = false

        let tint = AdaptiveTintView(frame: .zero)
        tint.translatesAutoresizingMaskIntoConstraints = false
        content.addSubview(tint)
        content.addSubview(root)
        NSLayoutConstraint.activate([
            tint.leadingAnchor.constraint(equalTo: content.leadingAnchor),
            tint.trailingAnchor.constraint(equalTo: content.trailingAnchor),
            tint.topAnchor.constraint(equalTo: content.topAnchor),
            tint.bottomAnchor.constraint(equalTo: content.bottomAnchor),
            root.leadingAnchor.constraint(equalTo: content.leadingAnchor, constant: 22),
            root.trailingAnchor.constraint(equalTo: content.trailingAnchor, constant: -22),
            root.topAnchor.constraint(equalTo: content.safeAreaLayoutGuide.topAnchor, constant: 14),
            root.bottomAnchor.constraint(equalTo: content.bottomAnchor, constant: -18),
        ])
        window.contentView = content

        content.layoutSubtreeIfNeeded()
        let fittingHeight = content.fittingSize.height
        window.setContentSize(NSSize(width: 420, height: min(540, max(210, fittingHeight))))
        window.center()
        window.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)

        if let first = request.accounts.first, let field = fields[first] {
            window.makeFirstResponder(field)
            field.setFocused(true)
        }
    }

    private func makeHeader(width: CGFloat) -> NSView {
        let row = NSStackView()
        row.orientation = .horizontal
        row.alignment = .top
        row.spacing = 9

        let symbol = NSImageView()
        symbol.image = NSImage(systemSymbolName: "lock.fill", accessibilityDescription: "Lock")
        symbol.symbolConfiguration = NSImage.SymbolConfiguration(pointSize: 15, weight: .semibold)
        symbol.contentTintColor = .secondaryLabelColor
        symbol.translatesAutoresizingMaskIntoConstraints = false
        symbol.widthAnchor.constraint(equalToConstant: 18).isActive = true
        symbol.heightAnchor.constraint(equalToConstant: 18).isActive = true

        let explanation = request.message?.trimmingCharacters(in: .whitespacesAndNewlines)
        let message = wrappingLabel(
            explanation?.isEmpty == false ? explanation! : "Paste the requested secrets.",
            width: width - 27
        )
        message.font = .systemFont(ofSize: 13.5, weight: .medium)
        message.textColor = .labelColor
        message.maximumNumberOfLines = 3

        row.addArrangedSubview(symbol)
        row.addArrangedSubview(message)
        row.widthAnchor.constraint(equalToConstant: width).isActive = true
        return row
    }

    private func makeFieldStack(width: CGFloat) -> NSStackView {
        let stack = NSStackView()
        stack.orientation = .vertical
        stack.alignment = .leading
        stack.spacing = 12
        stack.translatesAutoresizingMaskIntoConstraints = false

        for account in request.accounts {
            let group = NSStackView()
            group.orientation = .vertical
            group.alignment = .leading
            group.spacing = 5

            let label = NSTextField(labelWithString: account)
            label.font = .monospacedSystemFont(ofSize: 11, weight: .medium)
            label.textColor = .secondaryLabelColor

            let field = FideliusSecureTextField(frame: .zero)
            field.placeholderString = "Paste secret"
            field.font = .systemFont(ofSize: 13)
            field.delegate = self
            field.translatesAutoresizingMaskIntoConstraints = false
            field.widthAnchor.constraint(equalToConstant: width).isActive = true
            fields[account] = field

            group.addArrangedSubview(label)
            group.addArrangedSubview(field)
            stack.addArrangedSubview(group)
        }

        stack.widthAnchor.constraint(equalToConstant: width).isActive = true
        return stack
    }

    private func makeAutoDeleteRow() -> NSView {
        let row = NSStackView()
        row.orientation = .horizontal
        row.alignment = .centerY
        row.spacing = 6

        let icon = NSImageView()
        icon.image = NSImage(systemSymbolName: "timer", accessibilityDescription: "Auto-delete")
        icon.symbolConfiguration = NSImage.SymbolConfiguration(pointSize: 10, weight: .regular)
        icon.contentTintColor = .tertiaryLabelColor
        icon.translatesAutoresizingMaskIntoConstraints = false
        icon.widthAnchor.constraint(equalToConstant: 12).isActive = true
        icon.heightAnchor.constraint(equalToConstant: 12).isActive = true

        let label = NSTextField(labelWithString: "Auto-delete in \(request.autoDeleteLabel)")
        label.font = .systemFont(ofSize: 10.5)
        label.textColor = .tertiaryLabelColor

        row.addArrangedSubview(icon)
        row.addArrangedSubview(label)
        return row
    }

    private func makeButtons(width: CGFloat) -> NSView {
        let row = NSStackView()
        row.orientation = .horizontal
        row.alignment = .centerY
        row.spacing = 8

        let spacer = NSView()
        spacer.translatesAutoresizingMaskIntoConstraints = false
        row.addArrangedSubview(spacer)

        let cancelButton = NSButton(title: "Cancel", target: self, action: #selector(cancel(_:)))
        cancelButton.keyEquivalent = "\u{1b}"
        cancelButton.bezelStyle = .rounded
        row.addArrangedSubview(cancelButton)

        let saveButton = NSButton(
            title: request.accounts.count == 1 ? "Save Secret" : "Save Secrets",
            target: self,
            action: #selector(save(_:))
        )
        saveButton.keyEquivalent = "\r"
        saveButton.bezelStyle = .rounded
        saveButton.isEnabled = false
        self.saveButton = saveButton
        row.addArrangedSubview(saveButton)

        row.widthAnchor.constraint(equalToConstant: width).isActive = true
        return row
    }

    private func wrappingLabel(_ text: String, width: CGFloat) -> NSTextField {
        let label = NSTextField(wrappingLabelWithString: text)
        label.font = .systemFont(ofSize: 12.5)
        label.maximumNumberOfLines = 3
        label.preferredMaxLayoutWidth = width
        return label
    }

    private func buildMenus() {
        let mainMenu = NSMenu()

        let appItem = NSMenuItem()
        mainMenu.addItem(appItem)
        let appMenu = NSMenu()
        appMenu.addItem(withTitle: "Quit Fidelius", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        appItem.submenu = appMenu

        let editItem = NSMenuItem()
        mainMenu.addItem(editItem)
        let editMenu = NSMenu(title: "Edit")
        editMenu.addItem(withTitle: "Undo", action: Selector(("undo:")), keyEquivalent: "z")
        editMenu.addItem(NSMenuItem.separator())
        editMenu.addItem(withTitle: "Cut", action: #selector(NSText.cut(_:)), keyEquivalent: "x")
        editMenu.addItem(withTitle: "Copy", action: #selector(NSText.copy(_:)), keyEquivalent: "c")
        editMenu.addItem(withTitle: "Paste", action: #selector(NSText.paste(_:)), keyEquivalent: "v")
        editMenu.addItem(withTitle: "Select All", action: #selector(NSText.selectAll(_:)), keyEquivalent: "a")
        editItem.submenu = editMenu

        NSApp.mainMenu = mainMenu
    }
}
