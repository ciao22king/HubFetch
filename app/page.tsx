const features = [
  { number: "01", title: "Clone senza attriti", text: "GitHub, GitLab, SSH e repository self-hosted. Incolla l’URL e lascia fare a GoFetch." },
  { number: "02", title: "Cronologia intelligente", text: "Ritrova i progetti già scaricati, aprili al volo o aggiornali con un fast-forward sicuro." },
  { number: "03", title: "Build automatiche", text: "Riconosce Go, Rust, Node, Python, Java e Make. Dopo il clone, verifica che tutto funzioni." },
]

const commands = [
  ["$", "gofetch"],
  ["→", "Paste repository URL"],
  ["→", "Choose destination"],
  ["✓", "Cloned and built successfully"],
]

export default function Home() {
  return (
    <main>
      <nav className="nav shell">
        <a className="brand" href="#top" aria-label="GoFetch home"><span className="brand-mark">↳</span> gofetch</a>
        <div className="nav-links"><a href="#features">Features</a><a href="#workflow">How it works</a><a href="https://github.com/ciao22king/GoFetch" target="_blank" rel="noreferrer">GitHub ↗</a></div>
        <a className="nav-cta" href="#install">Install <span>→</span></a>
      </nav>

      <section className="hero shell" id="top">
        <div className="hero-copy">
          <p className="eyebrow"><span className="status-dot" /> Open source · v0.6.0</p>
          <h1>From repo URL<br /><em>to ready-to-code.</em></h1>
          <p className="hero-text">GoFetch is the calm, keyboard-first way to clone, build and revisit Git repositories. No incantations. Just ship.</p>
          <div className="hero-actions"><a className="button button-primary" href="#install">Start fetching <span>↗</span></a><a className="text-link" href="https://github.com/ciao22king/GoFetch" target="_blank" rel="noreferrer">Explore the source <span>↗</span></a></div>
          <p className="support-line">Works on <strong>macOS</strong>, <strong>Linux</strong> and <strong>Windows</strong></p>
        </div>
        <div className="terminal-wrap" aria-label="GoFetch terminal preview">
          <div className="terminal">
            <div className="terminal-top"><div className="window-dots"><span /><span /><span /></div><span className="terminal-title">gofetch — ~/Code</span><span className="terminal-version">v0.6.0</span></div>
            <div className="terminal-body"><div className="terminal-heading"><span className="prompt-symbol">↳</span><span> GoFetch</span><small>FETCH WHAT’S NEXT</small></div><div className="terminal-rule" />
              <p className="terminal-label">Recent fetches</p><div className="repo-row selected"><span className="repo-icon">◆</span><span>vercel / next.js</span><span className="repo-meta">2m ago</span></div><div className="repo-row"><span className="repo-icon">◇</span><span>charmbracelet / bubbletea</span><span className="repo-meta">yesterday</span></div><div className="repo-row"><span className="repo-icon">◇</span><span>ciao22king / GoFetch</span><span className="repo-meta">last week</span></div><div className="terminal-divider" /><div className="terminal-hint"><span className="key">n</span> new fetch <span className="key">u</span> update <span className="key">/</span> search <span className="key">q</span> quit</div>
            </div>
          </div><div className="terminal-shadow" />
        </div>
      </section>

      <section className="ticker"><div className="ticker-inner"><span>BUILT FOR THE TERMINAL</span><span className="ticker-dot">✦</span><span>LESS FRICTION, MORE FLOW</span><span className="ticker-dot">✦</span><span>OPEN SOURCE TOOLING</span><span className="ticker-dot">✦</span><span>BUILT FOR THE TERMINAL</span></div></section>

      <section className="features shell" id="features"><div className="section-intro"><p className="eyebrow">The better git clone</p><h2>Small tool.<br /><em>Big momentum.</em></h2></div><div className="feature-grid">{features.map((feature) => <article className="feature-card" key={feature.number}><span className="feature-number">{feature.number}</span><h3>{feature.title}</h3><p>{feature.text}</p><span className="feature-arrow">↗</span></article>)}</div></section>

      <section className="workflow shell" id="workflow"><div className="workflow-copy"><p className="eyebrow">A focused flow</p><h2>Less setup.<br /><em>More starting.</em></h2><p>Every decision is visible, every action is reversible. GoFetch keeps the terminal close and the busywork out of the way.</p><a className="text-link" href="#install">See the commands <span>→</span></a></div><div className="command-list">{commands.map(([symbol, text], index) => <div className={`command ${index === 3 ? "complete" : ""}`} key={text}><span className="command-symbol">{symbol}</span><span>{text}</span>{index === 3 && <span className="command-check">●</span>}</div>)}</div></section>

      <section className="install shell" id="install"><div><p className="eyebrow">Ready when you are</p><h2>Start with one<br /><em>simple command.</em></h2></div><div className="install-box"><p>Install the latest version</p><code><span>$</span> go install github.com/ciao22king/GoFetch@latest</code><a className="button button-light" href="https://github.com/ciao22king/GoFetch" target="_blank" rel="noreferrer">View on GitHub <span>↗</span></a></div></section>

      <footer className="footer shell"><a className="brand" href="#top"><span className="brand-mark">↳</span> gofetch</a><p>Clone calmly. Build confidently.</p><p>© 2025 GoFetch · MIT License</p></footer>
    </main>
  )
}
