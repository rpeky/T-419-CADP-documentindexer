iregex to parse shakesperean text to individual tokens, to include words with apostrophe like isn't as a token

Use a pattern that treats apostrophes inside a word as part of the token, but not leading/trailing punctuation.

Good default (handles isn’t / isn't / o’er / rock’n’roll)
(?i)\b[[:alpha:]]+(?:['’][[:alpha:]]+)*\b


['’] accepts both straight ' and curly ’ (common in nicer editions).

The (?: ... )* allows multiple apostrophe segments (e.g., rock’n’roll).

If you want to include numbers too (e.g., “2nd”, “’tis” still excluded)
(?i)\b[[:alnum:]]+(?:['’][[:alnum:]]+)*\b

If your regex engine doesn’t support POSIX classes ([[:alpha:]])

Use ASCII letters:

(?i)\b[a-z]+(?:['’][a-z]+)*\b


If you tell me what language/engine you’re using (Go, Python, PCRE, RE2), I can give the exact snippet for iterating matches and any engine-specific tweaks.

in golang

In Go you’re using RE2, so keep it RE2-compatible (no lookbehind, no \p{...} shortcuts unless you use Unicode classes explicitly).

Regex (ASCII letters + internal apostrophes)
re := regexp.MustCompile(`(?i)\b[a-z]+(?:['’][a-z]+)*\b`)

Tokenize example
package main

import (
	"fmt"
	"regexp"
)

func main() {
	text := "Isn't this O’er the hill? rock’n’roll; 'tis fine."
	re := regexp.MustCompile(`(?i)\b[a-z]+(?:['’][a-z]+)*\b`)
	toks := re.FindAllString(text, -1)
	fmt.Println(toks)
}


Output will include: Isn't, O’er, rock’n’roll, tis (note: leading apostrophe is dropped because we require starting with a letter).

If you want to keep leading-apostrophe forms like 'tis, 'twas

Use an optional leading apostrophe, but still require letters:

re := regexp.MustCompile(`(?i)\b['’]?[a-z]+(?:['’][a-z]+)*\b`)


That will tokenize 'tis as 'tis (including the apostrophe).

explain the regex used

Sure. I’ll explain this one (the “good default”):

(?i)\b[a-z]+(?:['’][a-z]+)*\b

(?i)

Case-insensitive flag. So it matches Isn't, ISN'T, isn't, etc., without you lowercasing first.

\b … \b

Word boundaries at the start and end. This makes sure we start/end on a “word edge” so punctuation like , . ! ? doesn’t get included.

[a-z]+

“At least one ASCII letter.”
This forces the token to start with letters, so you don’t accidentally treat ' or punctuation as a token by itself.

(?: ... )*

A non-capturing group repeated zero or more times.

Non-capturing (?:...) just groups without creating a capture group (cheaper/cleaner).

* means you can have none, one, or many of these extra segments.

['’][a-z]+

One apostrophe (either straight ' or curly ’), followed by at least one letter.

This is the key to “keep contractions as one token”:

isn't matches as: isn + (' + t)

o’er matches as: o + (’ + er)

rock’n’roll matches as: rock + (’+n) + (’+roll) because the group can repeat.

What it won’t match (by design)

Leading apostrophe forms like 'tis as a single token (it would return tis), because the token must start with [a-z].

Words with hyphens as one token (well-being becomes well and being). If you want hyphens kept too, we can extend it.

If you want, tell me whether you want to preserve 'tis and hyphenated words, and I’ll give the adjusted Go regex.

sure preserve the 'tis and hyphenated words

Use this in Go (RE2):

(?i)\b['’]?[a-z]+(?:['’][a-z]+)*(?:-[a-z]+(?:['’][a-z]+)*)*\b

What it matches

Leading apostrophe forms: 'tis, 'twas

Contractions/internal apostrophes: isn't, o’er, rock’n’roll

Hyphenated compounds (including multiple hyphens): well-being, mother-in-law

Hyphenated + apostrophes: jack-o’-lantern, don’t-care

How it works (piece by piece)

(?i) : case-insensitive.

\b : start at a word boundary.

['’]? : optional leading apostrophe (straight or curly). This is what allows 'tis.

[a-z]+ : the first “word chunk” must start with letters.

(?:['’][a-z]+)* : zero or more apostrophe + letters segments inside the chunk
(so isn't stays one token; rock’n’roll also works).

(?: ... )* after that is for hyphenated extensions:

(?:-[a-z]+(?:['’][a-z]+)*)*

Each hyphen part starts with -, then letters, then optional apostrophe segments.

Repeats so mother-in-law stays one token.

Final \b : end at a word boundary so trailing punctuation isn’t included.

Go snippet
re := regexp.MustCompile(`(?i)\b['’]?[a-z]+(?:['’][a-z]+)*(?:-[a-z]+(?:['’][a-z]+)*)*\b`)
tokens := re.FindAllString(text, -1)


If your Shakespeare text includes non-ASCII letters (rare, but can happen in some editions), we can switch [a-z] to a Unicode-letter class using Go’s \p{L}.

i want the words, i dont want the shite spaces

If you use FindAllString, you’ll get only the matched word tokens (no spaces). Spaces aren’t part of the match unless your regex includes \s or .* around it.

Use this:

re := regexp.MustCompile(`(?i)\b['’]?[a-z]+(?:['’][a-z]+)*(?:-[a-z]+(?:['’][a-z]+)*)*\b`)
tokens := re.FindAllString(text, -1) // <-- words only, no whitespace


Example:

package main

import (
	"fmt"
	"regexp"
)

func main() {
	text := "  'Tis  well-being, isn't it?  rock’n’roll!\n"
	re := regexp.MustCompile(`(?i)\b['’]?[a-z]+(?:['’][a-z]+)*(?:-[a-z]+(?:['’][a-z]+)*)*\b`)
	fmt.Printf("%q\n", re.FindAllString(text, -1))
}


Output:

["'Tis" "well-being" "isn't" "it" "rock’n’roll"]


If you’re currently seeing spaces in results, you likely used something like FindAllStringSubmatch with a pattern that captures whitespace, or your regex includes \s somewhere—paste your current regex + the code line you’re using and I’ll point to the exact culprit.

i am using regexp.mustompile.split

Ah—Split works the opposite way: it returns the stuff between matches, so if you split on “words”, you’ll get the whitespace/punctuation (“shite spaces”) as elements.

Do this instead:

Best: extract tokens (don’t split)
re := regexp.MustCompile(`(?i)\b['’]?[a-z]+(?:['’][a-z]+)*(?:-[a-z]+(?:['’][a-z]+)*)*\b`)
tokens := re.FindAllString(text, -1)


That’s the idiomatic “tokenizer” approach in Go.

If you insist on Split: split on NON-words, then filter empties

Split on delimiters (anything that’s not part of your token), so the result pieces are words.

delim := regexp.MustCompile(`(?i)[^a-z'’-]+`) // anything not letter/apostrophe/hyphen
parts := delim.Split(text, -1)

// filter empties
tokens := parts[:0]
for _, p := range parts {
	if p != "" {
		tokens = append(tokens, p)
	}
}

Notes

This keeps ' / ’ / - inside tokens.

It can still produce weird tokens if the text has sequences like --- or ' alone; the filter helps, but FindAllString is cleaner and more precise.

If you show me your exact Split regex, I can rewrite it into the right “delimiter split” form for your case.
