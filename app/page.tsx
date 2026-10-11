const benefits = [
  { number: "01", title: "Trova e scarica", text: "Incolla un URL GitHub, GitLab o SSH. HubFetch prepara la cartella giusta senza comandi da ricordare." },
  { number: "02", title: "Riprendi il contesto", text: "La cronologia dei repository resta a portata di mano: aggiorna, riapri e torna al lavoro in pochi secondi." },
  { number: "03", title: "Verifica subito", text: "Riconosce il progetto e avvia i passaggi di build. Sai immediatamente se il codice è pronto." },
]

const workflow = ["Incolla il repository", "Scegli dove salvarlo", "HubFetch clona e configura", "Inizia a costruire"]

export default function Home() {
  return (
    <main>
      <nav className="nav shell">
        <a className="brand" href="#top" aria-label="HubFetch home"><span className="brand-mark">↳</span> hubfetch</a>
        <div className="nav-links"><a href="#perche">Perché HubFetch</a><a href="#come-funziona">Come funziona</a><a href="#faq">FAQ</a><a href="https://github.com/ciao22king/HubFetch" target="_blank" rel="noreferrer">GitHub ↗</a></div>
        <a className="nav-cta" href="#install">Provalo <span>→</span></a>
      </nav>

      <section className="hero shell" id="top">
        <div className="hero-copy">
          <p className="eyebrow"><span className="status-dot" /> Open source · per chi costruisce</p>
          <h1>Dal repository<br /><em>al codice.</em></h1>
          <p className="hero-text">HubFetch rende semplice iniziare a lavorare su un progetto Git. Scarica, prepara e ritrova i tuoi repository senza interrompere il tuo flusso.</p>
          <div className="hero-actions"><a className="button button-primary" href="#install">Inizia ora <span>↗</span></a><a className="text-link" href="https://github.com/ciao22king/HubFetch" target="_blank" rel="noreferrer">Guarda il codice <span>↗</span></a></div>
          <p className="support-line">Funziona su <strong>macOS</strong>, <strong>Linux</strong> e <strong>Windows</strong></p>
        </div>
        <div className="terminal-wrap" aria-label="Anteprima del terminale HubFetch">
          <div className="terminal"><div className="terminal-top"><div className="window-dots"><span /><span /><span /></div><span className="terminal-title">hubfetch — ~/Code</span><span className="terminal-version">READY</span></div><div className="terminal-body"><div className="terminal-heading"><span className="prompt-symbol">↳</span><span> HubFetch</span><small>FETCH WHAT&apos;S NEXT</small></div><div className="terminal-rule" /><p className="terminal-label">Repository recenti</p><div className="repo-row selected"><span className="repo-icon">◆</span><span>vercel / next.js</span><span className="repo-meta">2m fa</span></div><div className="repo-row"><span className="repo-icon">◇</span><span>charmbracelet / bubbletea</span><span className="repo-meta">ieri</span></div><div className="repo-row"><span className="repo-icon">◇</span><span>ciao22king / HubFetch</span><span className="repo-meta">la scorsa settimana</span></div><div className="terminal-divider" /><div className="terminal-hint"><span className="key">n</span> nuovo <span className="key">u</span> aggiorna <span className="key">/</span> cerca <span className="key">q</span> esci</div></div></div><div className="terminal-shadow" />
        </div>
      </section>

      <section className="proof"><div className="shell proof-grid"><div><strong>1</strong><span>comando<br />da imparare</span></div><div><strong>0</strong><span>configurazioni<br />inutili</span></div><div><strong>∞</strong><span>progetti<br />da esplorare</span></div><p>Il modo più diretto per passare dall&apos;idea di un progetto al primo commit.</p></div></section>

      <section className="why shell" id="perche"><div className="section-intro"><p className="eyebrow">Perché usarlo</p><h2>Il tuo tempo è<br /><em>per il codice.</em></h2><p className="section-lead">Il primo contatto con un repository non dovrebbe essere una caccia al tesoro tra comandi, cartelle e istruzioni sparse.</p></div><div className="feature-grid">{benefits.map((benefit) => <article className="feature-card" key={benefit.number}><span className="feature-number">{benefit.number}</span><h3>{benefit.title}</h3><p>{benefit.text}</p><span className="feature-arrow">↗</span></article>)}</div></section>

      <section className="workflow shell" id="come-funziona"><div className="workflow-copy"><p className="eyebrow">Come funziona</p><h2>Quattro passi.<br /><em>Nessun attrito.</em></h2><p>Un flusso pensato per sviluppatori, studenti e team che vogliono esplorare più codice e perdere meno tempo nella preparazione.</p><a className="text-link" href="#install">Installa HubFetch <span>→</span></a></div><div className="steps">{workflow.map((step, index) => <div className="step" key={step}><span className="step-number">0{index + 1}</span><span>{step}</span>{index < workflow.length - 1 && <span className="step-line" />}</div>)}</div></section>

      <section className="audience shell"><div className="audience-heading"><p className="eyebrow">Pensato per il tuo flusso</p><h2>Un piccolo<br /><em>superpotere.</em></h2></div><div className="audience-list"><div><span>01</span><h3>Per chi esplora</h3><p>Prova nuovi progetti senza sporcare il tuo ambiente con passaggi manuali.</p></div><div><span>02</span><h3>Per chi collabora</h3><p>Porta un repository dal team al tuo editor con un percorso sempre chiaro.</p></div><div><span>03</span><h3>Per chi impara</h3><p>Concentrati su come funziona il codice, non su dove finisce la cartella.</p></div></div></section>

      <section className="install shell" id="install"><div><p className="eyebrow">Pronto a provarlo?</p><h2>Comincia da<br /><em>una riga.</em></h2><p className="install-copy">HubFetch è open source, gratuito e fatto per restare fuori dal tuo percorso.</p></div><div className="install-box"><p>Installa l&apos;ultima versione</p><code><span>$</span> go install github.com/ciao22king/HubFetch@latest</code><a className="button button-light" href="https://github.com/ciao22king/HubFetch" target="_blank" rel="noreferrer">Vai su GitHub <span>↗</span></a></div></section>

      <section className="faq shell" id="faq"><div><p className="eyebrow">Domande rapide</p><h2>Prima di<br /><em>iniziare.</em></h2></div><div className="faq-list"><details open><summary>Cos&apos;è HubFetch?</summary><p>È uno strumento da terminale open source per clonare, organizzare e aggiornare repository Git in modo più semplice.</p></details><details><summary>Devo cambiare il mio workflow?</summary><p>No. HubFetch si affianca a Git e ti aiuta nella parte ripetitiva: trovare il progetto, scaricarlo e prepararlo.</p></details><details><summary>È davvero gratuito?</summary><p>Sì. Il progetto è pubblico su GitHub con licenza MIT. Puoi usarlo, studiarlo e contribuire.</p></details></div></section>

      <footer className="footer shell"><a className="brand" href="#top"><span className="brand-mark">↳</span> hubfetch</a><p>Più tempo per costruire. Meno tempo per preparare.</p><p>© 2026 HubFetch · MIT License</p></footer>
    </main>
  )
}
