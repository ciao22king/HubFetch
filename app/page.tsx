const benefits = [
  { number: "01", title: "Start in seconds", text: "Paste a GitHub, GitLab, or SSH URL and get straight to the code. No scattered commands, no setup maze." },
  { number: "02", title: "Stay organized", text: "Keep every project in one predictable place. Reopen recent repositories and pick up exactly where you left off." },
  { number: "03", title: "Built for momentum", text: "A focused terminal workflow that makes cloning and updating feel as quick as thinking about the next idea." },
]

const workflow = ["Paste a repository URL", "Choose your destination", "HubFetch clones and prepares it", "Open the project and build"]

export default function Home() {
  return (
    <main>
      <nav className="nav shell">
        <a className="brand" href="#top" aria-label="HubFetch home"><span className="brand-mark">↳</span> hubfetch</a>
        <div className="nav-links"><a href="#why">Why HubFetch</a><a href="#workflow">How it works</a><a href="#faq">FAQ</a><a href="https://github.com/ciao22king/HubFetch" target="_blank" rel="noreferrer">GitHub ↗</a></div>
        <a className="nav-cta" href="#install">Try it <span>→</span></a>
      </nav>

      <section className="hero shell" id="top">
        <div className="hero-copy">
          <p className="eyebrow"><span className="status-dot" /> Open source · for builders</p>
          <h1>Get to the code.<br /><em>Not the setup.</em></h1>
          <p className="hero-text">HubFetch is the clean, focused way to clone, organize, and revisit Git repositories. Less terminal friction means more time building things that matter.</p>
          <div className="hero-actions"><a className="button button-primary" href="#install">Get started <span>↗</span></a><a className="text-link" href="https://github.com/ciao22king/HubFetch" target="_blank" rel="noreferrer">View the code <span>↗</span></a></div>
          <p className="support-line">Works on <strong>macOS</strong>, <strong>Linux</strong>, and <strong>Windows</strong></p>
        </div>
        <div className="terminal-wrap" aria-label="HubFetch terminal preview">
          <div className="terminal"><div className="terminal-top"><div className="window-dots"><span /><span /><span /></div><span className="terminal-title">hubfetch — ~/Code</span><span className="terminal-version">READY</span></div><div className="terminal-body"><div className="terminal-heading"><span className="prompt-symbol">↳</span><span> HubFetch</span><small>FETCH WHAT&apos;S NEXT</small></div><div className="terminal-rule" /><p className="terminal-label">Recent repositories</p><div className="repo-row selected"><span className="repo-icon">◆</span><span>vercel / next.js</span><span className="repo-meta">2m ago</span></div><div className="repo-row"><span className="repo-icon">◇</span><span>charmbracelet / bubbletea</span><span className="repo-meta">yesterday</span></div><div className="repo-row"><span className="repo-icon">◇</span><span>ciao22king / HubFetch</span><span className="repo-meta">last week</span></div><div className="terminal-divider" /><div className="terminal-hint"><span className="key">n</span> new <span className="key">u</span> update <span className="key">/</span> search <span className="key">q</span> quit</div></div></div><div className="terminal-shadow" />
        </div>
      </section>

      <section className="proof"><div className="shell proof-grid"><div><strong>1</strong><span>command<br />to learn</span></div><div><strong>0</strong><span>unnecessary<br />configurations</span></div><div><strong>∞</strong><span>projects<br />to explore</span></div><p>One simple starting point for every project you want to understand, improve, or make your own.</p></div></section>

      <section className="why shell" id="why"><div className="section-intro"><p className="eyebrow">Why developers use it</p><h2>Your time is<br /><em>for the code.</em></h2><p className="section-lead">The first five minutes with a repository should not be spent hunting for folders, repeating commands, or wondering what happened to your last project.</p></div><div className="feature-grid">{benefits.map((benefit) => <article className="feature-card" key={benefit.number}><span className="feature-number">{benefit.number}</span><h3>{benefit.title}</h3><p>{benefit.text}</p><span className="feature-arrow">↗</span></article>)}</div></section>

      <section className="statement"><div className="shell statement-inner"><p className="eyebrow">The HubFetch mindset</p><blockquote>“Make the path to great code feel <em>obvious.</em>”</blockquote><p>HubFetch handles the repetitive first step so you can focus on what you actually came to do: learn, collaborate, and ship.</p></div></section>

      <section className="workflow shell" id="workflow"><div className="workflow-copy"><p className="eyebrow">How it works</p><h2>Four steps.<br /><em>Zero friction.</em></h2><p>A clear flow for developers, students, and teams who want to explore more code and spend less time getting ready.</p><a className="text-link" href="#install">Install HubFetch <span>→</span></a></div><div className="steps">{workflow.map((step, index) => <div className="step" key={step}><span className="step-number">0{index + 1}</span><span>{step}</span>{index < workflow.length - 1 && <span className="step-line" />}</div>)}</div></section>

      <section className="audience shell"><div className="audience-heading"><p className="eyebrow">Made for every kind of builder</p><h2>A small<br /><em>superpower.</em></h2></div><div className="audience-list"><div><span>01</span><h3>Explore without clutter</h3><p>Try new projects without filling your machine with mystery folders and forgotten clones.</p></div><div><span>02</span><h3>Collaborate with clarity</h3><p>Bring a team repository from URL to editor through a path everyone can understand.</p></div><div><span>03</span><h3>Learn by doing</h3><p>Spend your energy understanding how the code works, not where the folder ended up.</p></div></div></section>

      <section className="install shell" id="install"><div><p className="eyebrow">Ready to try it?</p><h2>Start with<br /><em>one line.</em></h2><p className="install-copy">HubFetch is open source, free, and designed to stay out of your way. Install it once and make every repository easier to reach.</p></div><div className="install-box"><p>Install the latest version</p><code><span>$</span> go install github.com/ciao22king/HubFetch@latest</code><a className="button button-light" href="https://github.com/ciao22king/HubFetch" target="_blank" rel="noreferrer">Go to GitHub <span>↗</span></a></div></section>

      <section className="faq shell" id="faq"><div><p className="eyebrow">Quick questions</p><h2>Before you<br /><em>begin.</em></h2></div><div className="faq-list"><details open><summary>What is HubFetch?</summary><p>It is an open source terminal tool for cloning, organizing, and updating Git repositories more simply.</p></details><details><summary>Does it replace Git?</summary><p>No. HubFetch works alongside Git and makes the repetitive parts of getting to a repository faster and more memorable.</p></details><details><summary>Who is it for?</summary><p>Anyone who works with code: curious learners, independent developers, and teams moving between multiple projects.</p></details><details><summary>Is it really free?</summary><p>Yes. The project is public on GitHub under the MIT license. Use it, study it, and contribute.</p></details></div></section>

      <footer className="footer shell"><a className="brand" href="#top"><span className="brand-mark">↳</span> hubfetch</a><p>More time building. Less time preparing.</p><p>© 2026 HubFetch · MIT License</p></footer>
    </main>
  )
}
