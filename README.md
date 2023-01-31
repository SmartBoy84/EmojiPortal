# EmojiPortal
Scrape latest set of emojis from Unicode library and use scraped emojis to convert images to a array of emojis 

\* This project unifies [Emojifier](https://github.com/SmartBoy84/Emojifier) and [EmojiScraper](https://github.com/SmartBoy84/EmojiScraper) + adds a TON more features

`[folderNames... cartridgeFiles... html internal] % [cart/list] {scale:int} [folderName]`  
`{...} % {emojify {escale:int (emoji scale)} {iscale:int (image scale)} {quality:int} [Source image] {target image}}`    

## Explanation
- In all of the following cases `src` can be `internal`, in which case the embedded cartridge is used - exclusion of any option assumes `internal` (must specify `%` though)
- If you don't specify a destination mode then it is assumed to be `cart`
- If you don't specify a destination folder then it is assumed to be `cart == cartridges` and `list == emojis`

## Examples 
### Scraping 
`./cartridges html:1`  
`./cartridges internal`  
`./emojiportal html % cart == ./emojipotatl % cart`   
`./emojiportal html % cart scale:85 cartridges`  
`./emojiportal cartridges/* % list scale:65 emojis`  

### Emojifying
`./emojiportal html % emojify iscale:0.5 escale:0.2 quality:75 in.png`  
`./emojiportal cartridges/Apple.png % emojify iscale:0.5 escale:0.2 quality:75 in.png`  