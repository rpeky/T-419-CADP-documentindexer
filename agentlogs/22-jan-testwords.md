Query: give me words to  from the collection of shakespheere plays that might be significant in a few but in a few other plays, like character names or terms?

Response:
Yep — you want medium document frequency tokens: show up in a cluster of related plays (2–5-ish), not everywhere.

Here are good probes (single tokens) you can paste into your REPL.

Clusters by “shared world / topic”
Roman / Republic / Empire cluster (Julius Caesar, Antony & Cleopatra, Coriolanus, Titus, sometimes Cymbeline)

caesar

octavius

triumvir

tribune

plebeian

senate

capitol

rome

tiber

legion

English histories cluster (Richard II/III, Henry IV/V/VI, King John, Henry VIII)

plantagenet

lancaster

york

bolingbroke

gloucester

warwick

somerset

norfolk

exeter

talbot

Falstaff / tavern / commoners cluster (Henry IV Pt 1+2, Merry Wives; sometimes Henry V mentions)

falstaff

hal

bardolph

pistol

nim

poins

mistress (can appear elsewhere too, but often spikes here)

hostess

Fairy / magic / folklore cluster (Midsummer, Tempest, Macbeth, sometimes Winter’s Tale)

fairy

sprite

witch

weird (as in “weird sisters” → Macbeth; also appears elsewhere a bit)

spell

charm

prophecy

Italy / city-state / courtly-feud cluster (Romeo+Juliet, Othello, Merchant, Two Gentlemen, Shrew)

venice

verona

mantua

padua

messina (Much Ado)

capulet

montague

iago (mostly Othello, but nice for sparsity)

“Institution / role” words that show up in a few plays (not all)

tribune (Roman plays)

herald (histories + a few tragedies)

banished (spikes in Romeo+Juliet and some histories)

friar (Romeo+Juliet + a couple others)

duchess (histories + some tragedies)

Quick practical tip

Character-name probes work best if your files include speech prefixes / dramatis personae (e.g., HAMLET. lines). If your corpus is “cleaned prose” without those, prefer the theme/institution probes like triumvir, plebeian, lancaster, capulet, venice.

If you tell me which play collection you’re indexing (MIT, Gutenberg, OpenSourceShakespeare, etc.), I can tailor a tighter list that’s guaranteed to hit “~2–4 plays” in that exact corpus.
