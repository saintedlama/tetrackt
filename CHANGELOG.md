# Changelog

## 1.0.0 (2026-04-13)


### Features

* Add a third osc/env/lfo bank ([92b35fa](https://github.com/saintedlama/tetrackt/commit/92b35fa23214534f4df89f9fd574a2951feb4e7d))
* add ADSR filter envelope support to synth and UI components ([a28e2f5](https://github.com/saintedlama/tetrackt/commit/a28e2f5e98bc616aae1dcd8841c607728055ecc2))
* add detune functionality to oscillators and update related tests ([d5ded15](https://github.com/saintedlama/tetrackt/commit/d5ded150fa17faffb1387f085c38a125c35f79e5))
* add detune modulation support to oscillators ([547d04d](https://github.com/saintedlama/tetrackt/commit/547d04da237dced5a33d5bb5812d5f313fe2d9d4))
* add file dialog for saving and loading songs in JSON format ([992a4f4](https://github.com/saintedlama/tetrackt/commit/992a4f48e6e7b856207f28d62608ae4d0903506b))
* add initial logo ([f399fc6](https://github.com/saintedlama/tetrackt/commit/f399fc69101cdb676950e94dc70502badcb91ac7))
* Add low-frequency oscillator (LFO) implementation and UI components ([42329c8](https://github.com/saintedlama/tetrackt/commit/42329c89cb2cbe54fdbc5057a8c912595e2602dc))
* add MCP server mode for tracker editing ([#12](https://github.com/saintedlama/tetrackt/issues/12)) ([5366439](https://github.com/saintedlama/tetrackt/commit/5366439eec28bcb06d8b9187b95e15b39cae8bb0))
* add portamento support for smooth frequency transitions in synth and oscillator ([d23c461](https://github.com/saintedlama/tetrackt/commit/d23c461a2c951ef673556ac55136d14076ba63ab))
* add presets for classic drum machines ([f9b8bf7](https://github.com/saintedlama/tetrackt/commit/f9b8bf7354cedd18d0931d453fe4fe4db6b5362c))
* add wavetable osc ([e58f37f](https://github.com/saintedlama/tetrackt/commit/e58f37f405f4da994279c532c1d80b4613357a2a))
* continuous tick option and enhance row effects dialog with ticks display ([d541bc7](https://github.com/saintedlama/tetrackt/commit/d541bc78d6d4a627c17394ca2e813f77e2a7b4d8))
* enhance arpeggio functionality with per-row tick count and update UI components ([ce55f23](https://github.com/saintedlama/tetrackt/commit/ce55f23c02287feb3d12abacbb6bf67806fe4718))
* enhance envelope handling with time.Duration and update serialization for persistence ([3addb58](https://github.com/saintedlama/tetrackt/commit/3addb58eaa0cc5e97d9f55bc346f774d1dcbad2b))
* Enhance input profile support and update keybindings in documentation and UI ([dc92ee3](https://github.com/saintedlama/tetrackt/commit/dc92ee3eddf426a634d86bc2ed913fd1980ed6d1))
* enhance mixer functionality with portamento support ([0b71be1](https://github.com/saintedlama/tetrackt/commit/0b71be11183337d2b0112af8cf5e8d3af94b36d2))
* enhance mixer with panning, muting, and mode selection features ([894bc92](https://github.com/saintedlama/tetrackt/commit/894bc92357810018adb4c4bf69cb02ab7eabb60c))
* enhance synth patch metadata and improve UI, ship demo and quickstart modules ([6d41988](https://github.com/saintedlama/tetrackt/commit/6d4198826b15df617ec40320d09c95c9e5cf1a99))
* enhance UI navigation with grid-based panel movement and improve rendering consistency ([c62c9b8](https://github.com/saintedlama/tetrackt/commit/c62c9b8f7cd4ca7bcd9a283ef8880101db93012f))
* implement arpeggio preview and effect formatting in player and tracker UI ([d830b76](https://github.com/saintedlama/tetrackt/commit/d830b76c456a962795e905acb144cb70b778ec27))
* implement BPM adjustments and integrate into tracker settings panel ([e8a3d1d](https://github.com/saintedlama/tetrackt/commit/e8a3d1d7d874d9114970dc8765e85427788a298e))
* implement consistent help system ([98ba936](https://github.com/saintedlama/tetrackt/commit/98ba936e0b08a951b58bdb5b0ba9e3704ab38512))
* Implement modern keybinding profile and refactor navigation ([b0904eb](https://github.com/saintedlama/tetrackt/commit/b0904ebbd3a4366745773e6274267b215984b3c6))
* implement note-on/note-off gate model and per-row effects ([18deff8](https://github.com/saintedlama/tetrackt/commit/18deff8973aaa877867c9abdd565c48dfb0ec66c))
* Implement patch bank functionality and remove synth presets ([f1daeeb](https://github.com/saintedlama/tetrackt/commit/f1daeeb278aca80b4918a85ccff1d4f4d52100a8))
* implement patch system for synthesizer and update playback methods ([159d45e](https://github.com/saintedlama/tetrackt/commit/159d45ef35316af193734e58cada9efe67834a8b))
* implement portamento functionality for smooth pitch glides in synthesizer ([a746a42](https://github.com/saintedlama/tetrackt/commit/a746a42847b3208a7e2e5812ccdf610eb47412c3))
* implement UX improvement to move the cursor to the next line after setting a note ([caa7961](https://github.com/saintedlama/tetrackt/commit/caa796160b9b52581e0fcf99556da0aa71dcf82d))
* initial filter ([a8d12ab](https://github.com/saintedlama/tetrackt/commit/a8d12abd2b4f6b9ff874119c0cc192ac35f09182))
* introduce patchbank with presets ([6447147](https://github.com/saintedlama/tetrackt/commit/644714740aebee7b6b46eaf51fe993494a7ed34c))
* show MCP indicator if synth MCP is active ([40b8fa4](https://github.com/saintedlama/tetrackt/commit/40b8fa4c1aa8ca229538635a6b1823cf6195372f))
* **tracker:** implement inline effects editing and improve UX ([542f527](https://github.com/saintedlama/tetrackt/commit/542f5274c23c7b7c9f390da992e146373a5c9c75))
* Update mixer implementation to use independent volume controls for oscillators ([5fd00ce](https://github.com/saintedlama/tetrackt/commit/5fd00ce60e41404aab41b29c742009b59d628eac))


### Bug Fixes

* condense duplicate shortcuts in one help item ([9d76e5c](https://github.com/saintedlama/tetrackt/commit/9d76e5c82a846a4c8f2dd0777025aec9ef5ca56e))
* correct panel indexing and update UI layout for synth components ([211164c](https://github.com/saintedlama/tetrackt/commit/211164cd4205f0ba3ccee62cedea4a96e9df4607))
* repair rendering of the bar ([0b534e0](https://github.com/saintedlama/tetrackt/commit/0b534e0c4a5f080bc5bc62ca8c0a3750863e6492))
* set sustain minimum level never to 0 ([be8f879](https://github.com/saintedlama/tetrackt/commit/be8f87935f4c6f2abe3cd5b6440ea5b07cdefe77))
