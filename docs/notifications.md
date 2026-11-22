# Messages

The messages panel (`M`) holds notices aghelm raises about your own install.
It has no network source: aghelm reads no release catalog and no message feed.
Today the one notice is the HEADS UP that appears when a `config.toml` from an
earlier version could not be carried into Settings.

## The messages panel

The panel is an inbox. Messages sit in a list and the selected one opens beside
it.

The list groups messages by kind. A group with nothing in it is left out. Each
message takes two lines: the first holds a glyph for its kind and the title,
the second is dim and holds its subtitle. A message that has not been read yet
is bold and ends in a `●`.

A terminal 124 columns wide or more splits the panel, as long as it is tall
enough to show the whole list. The list sits on the left and the message on the
right, wrapped at the width of its pane. A narrower terminal, or a wide but
short one, puts the list above the message, one line per row.
Compatible glyph mode draws an ASCII stand-in for each kind, unread, highlight
and call-to-action glyph. The `↑↓` key hint and the `·` separator stay as
they are, and the rules and the scrollbar follow the app's compatible chrome.

A message shows a label chip for its kind, its subtitle, the title and the
body. When it has a link, a bar below the body in the kind's color reads `↵`
and the action text, with the link's host at its right.

### Keys

| Key | Action |
|---|---|
| `↑` `↓` or `k` `j` | Previous or next message |
| `pgup` `pgdn` `home` `end` | Scroll the message |
| `↵` | Open the message's link |
| `x` | Dismiss the message for good, except a pinned one |
| `esc` or `q` | Close |

The bottom edge lists the keys that apply to the selected message.

### Unread

A message is unread until it has been the selected message while the panel is
open. The set of read messages is stored beside the dismissed set. The top
edge of the panel reads `N unread` while any message is unread. The footer
chip reads `unread` while any message is unread and `messages` once all have
been read. It disappears when no message is left. Dismissing a message removes
it from the list.
