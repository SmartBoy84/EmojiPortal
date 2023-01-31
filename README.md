# EmojiPortal
Scrape latest set of emojis from Unicode library and use scraped emojis to convert images to a array of emojis 

\* This project unifies [Emojifier](https://github.com/SmartBoy84/Emojifier) and [EmojiScraper](https://github.com/SmartBoy84/EmojiScraper) + adds a TON more features

`[folderNames... cartridgeFiles... html] % [cart/list] {scale:int} [folderName]`

## Examples 
### Scraping
`./emojiportal` - scrapes all emojis from unicode (excluding modifiers) as cartridges into folder `./cartridges` 
`./cartridges html:1` - assumes dst name to be `cartridges`
`./cartridges cartridges/*` 
`./emojiportal html % cart == ./emojipotatl % cart` 
`./emojiportal html % cart scale:85 cartridges`  
`./emojiportal cartridges/* % list scale:65 emojis`

### Emojifying
`./emojiportal html % emojify iscale:0.5 escale:0.2 quality:75 in.png`
`./emojiportal cartridges/Apple.png % emojify iscale:0.5 escale:0.2 quality:75 in.png`
