import AppKit

/// Read-only, selectable message text. Stays out of the Tab loop so keyboard
/// focus moves between the secret fields and buttons only.
private final class MessageTextView: NSTextView {
    override var canBecomeKeyView: Bool { false }

    override func doCommand(by selector: Selector) {
        switch selector {
        case #selector(insertTab(_:)):
            window?.selectNextKeyView(self)
        case #selector(insertBacktab(_:)):
            window?.selectPreviousKeyView(self)
        default:
            super.doCommand(by: selector)
        }
    }
}

/// Builds the prompt's explanatory message. Short messages render exactly like
/// a wrapping label; long messages grow the window up to `maxHeight`, then
/// scroll inside a subtle inset well. Text is selectable and URLs are links.
enum MessageView {
    static let font = NSFont.systemFont(ofSize: 13.5, weight: .medium)
    static let wellInset = NSSize(width: 8, height: 6)

    /// Returns the message view and, when the text overflows `maxHeight`, the
    /// scroll view whose first line sits `wellInset.height` below the top.
    static func make(text: String, width: CGFloat, maxHeight: CGFloat) -> (view: NSView, scrollView: NSScrollView?) {
        let storage = NSTextStorage(attributedString: attributed(text))
        let layoutManager = NSLayoutManager()
        storage.addLayoutManager(layoutManager)
        let container = NSTextContainer(size: NSSize(width: width, height: .greatestFiniteMagnitude))
        container.lineFragmentPadding = 0
        container.widthTracksTextView = true
        layoutManager.addTextContainer(container)

        let textView = MessageTextView(frame: NSRect(x: 0, y: 0, width: width, height: 0), textContainer: container)
        textView.isEditable = false
        textView.isSelectable = true
        textView.isRichText = false
        textView.drawsBackground = false
        textView.textContainerInset = .zero
        textView.isVerticallyResizable = true
        textView.isHorizontallyResizable = false
        textView.autoresizingMask = [.width]
        textView.minSize = NSSize(width: 0, height: 0)
        textView.maxSize = NSSize(width: CGFloat.greatestFiniteMagnitude, height: .greatestFiniteMagnitude)
        textView.linkTextAttributes = [
            .foregroundColor: NSColor.linkColor,
            .cursor: NSCursor.pointingHand,
        ]
        textView.setAccessibilityLabel("Message")

        layoutManager.ensureLayout(for: container)
        let textHeight = ceil(layoutManager.usedRect(for: container).height)

        if textHeight <= maxHeight {
            textView.frame.size.height = textHeight
            textView.translatesAutoresizingMaskIntoConstraints = false
            NSLayoutConstraint.activate([
                textView.widthAnchor.constraint(equalToConstant: width),
                textView.heightAnchor.constraint(equalToConstant: textHeight),
            ])
            return (textView, nil)
        }

        // Overflow: inset the text inside a rounded well so the scroll region reads clearly.
        container.size = NSSize(width: width - wellInset.width * 2, height: .greatestFiniteMagnitude)
        textView.textContainerInset = wellInset

        let scrollView = NSScrollView()
        scrollView.drawsBackground = false
        scrollView.borderType = .noBorder
        scrollView.hasVerticalScroller = true
        scrollView.hasHorizontalScroller = false
        scrollView.autohidesScrollers = true
        scrollView.scrollerStyle = .overlay
        scrollView.verticalScrollElasticity = .allowed
        scrollView.translatesAutoresizingMaskIntoConstraints = false
        scrollView.documentView = textView
        textView.frame.size.width = width

        let well = InsetWellView()
        well.translatesAutoresizingMaskIntoConstraints = false
        well.addSubview(scrollView)
        NSLayoutConstraint.activate([
            scrollView.leadingAnchor.constraint(equalTo: well.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: well.trailingAnchor),
            scrollView.topAnchor.constraint(equalTo: well.topAnchor),
            scrollView.bottomAnchor.constraint(equalTo: well.bottomAnchor),
            well.widthAnchor.constraint(equalToConstant: width),
            well.heightAnchor.constraint(equalToConstant: maxHeight),
        ])
        return (well, scrollView)
    }

    private static func attributed(_ text: String) -> NSAttributedString {
        let paragraph = NSMutableParagraphStyle()
        paragraph.lineBreakMode = .byWordWrapping
        paragraph.lineSpacing = 1.5
        paragraph.paragraphSpacing = 3
        let result = NSMutableAttributedString(string: text, attributes: [
            .font: font,
            .foregroundColor: NSColor.labelColor,
            .paragraphStyle: paragraph,
        ])
        let range = NSRange(text.startIndex..., in: text)
        if let detector = try? NSDataDetector(types: NSTextCheckingResult.CheckingType.link.rawValue) {
            let source = text as NSString
            for match in detector.matches(in: text, options: [], range: range) {
                // Link only URLs the agent spelled out, not bare domains.
                let literal = source.substring(with: match.range).lowercased()
                guard let url = match.url, literal.hasPrefix("https://") || literal.hasPrefix("http://") else { continue }
                result.addAttribute(.link, value: url, range: match.range)
            }
        }
        return result
    }
}

/// Rounded, appearance-aware backdrop for the scrolling message.
private final class InsetWellView: NSView {
    override init(frame frameRect: NSRect) {
        super.init(frame: frameRect)
        wantsLayer = true
        layer?.cornerRadius = 7
        layer?.masksToBounds = true
        layer?.borderWidth = 1
        updateColors()
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        wantsLayer = true
        updateColors()
    }

    override func viewDidChangeEffectiveAppearance() {
        super.viewDidChangeEffectiveAppearance()
        updateColors()
    }

    private func updateColors() {
        guard let layer else { return }
        effectiveAppearance.performAsCurrentDrawingAppearance {
            layer.backgroundColor = NSColor.labelColor.withAlphaComponent(0.045).cgColor
            layer.borderColor = NSColor.separatorColor.cgColor
        }
    }
}
