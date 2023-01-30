# EmojiPortal
Scrape latest set of emojis from Unicode library

`[folderNames... cartridgeFiles... html] % [cart/list] {scale:int} [folderName]`

E.g.,
`./emojiportal` - scrapes all emojis from unicode (excluding modifiers) as cartridges into folder `./cartridges`  
`./emojiportal html % cart` 
`./emojiportal html % cart scale:85 cartridges`  
`./emojiportal cartridges/* % list scale:65 emojis`
