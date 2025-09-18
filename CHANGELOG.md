# Changelog

## [0.38.4](https://github.com/instill-ai/model-backend/compare/v0.38.3...v0.38.4) (2025-09-16)


### Miscellaneous

* **deps:** bump github.com/docker/docker from 26.1.5+incompatible to 28.0.0+incompatible ([#771](https://github.com/instill-ai/model-backend/issues/771)) ([5608f02](https://github.com/instill-ai/model-backend/commit/5608f02e5c284219e202f909df473b3c136a4974))

## [0.38.3](https://github.com/instill-ai/model-backend/compare/v0.38.2...v0.38.3) (2025-09-09)


### Miscellaneous

* **x:** update x package to v0.10.0-alpha ([#774](https://github.com/instill-ai/model-backend/issues/774)) ([15b412c](https://github.com/instill-ai/model-backend/commit/15b412c8bde55e8dc31b21b07f7cf8597c6c896b))

## [0.38.2](https://github.com/instill-ai/model-backend/compare/v0.38.1...v0.38.2) (2025-08-22)


### Miscellaneous

* **minio:** use new bucket name ([#772](https://github.com/instill-ai/model-backend/issues/772)) ([2ea7f85](https://github.com/instill-ai/model-backend/commit/2ea7f850f940b6e0e0d86e96c3e6c95f324f7010))

## [0.38.1](https://github.com/instill-ai/model-backend/compare/v0.38.0...v0.38.1) (2025-08-05)


### Features

* **temporal:** remove Temporal namespace initialization ([#769](https://github.com/instill-ai/model-backend/issues/769)) ([49e5f26](https://github.com/instill-ai/model-backend/commit/49e5f26a57c13b44704ec936dd889218dc620285))

## [0.38.0](https://github.com/instill-ai/model-backend/compare/v0.37.0...v0.38.0) (2025-07-31)


### Miscellaneous

* release v0.38.0 ([308ab73](https://github.com/instill-ai/model-backend/commit/308ab735aa82ff5d6f94250a4c052a40907ae28f))

## [0.37.0](https://github.com/instill-ai/model-backend/compare/v0.36.1...v0.37.0) (2025-07-16)


### Features

* **otel:** integrate OTEL using gRPC interceptor ([#763](https://github.com/instill-ai/model-backend/issues/763)) ([2b479db](https://github.com/instill-ai/model-backend/commit/2b479dbd8c26171c689f290960fea7fbac15bcf7))


### Miscellaneous

* release v0.37.0 ([91bca67](https://github.com/instill-ai/model-backend/commit/91bca67bf507eba116309507e290563fa02bcedc))


### Refactor

* **config:** remove db version from config file ([#761](https://github.com/instill-ai/model-backend/issues/761)) ([e0b0e8f](https://github.com/instill-ai/model-backend/commit/e0b0e8f6874f12421691b141d5a2e6f15acb3969))

## [0.36.1](https://github.com/instill-ai/model-backend/compare/v0.36.0...v0.36.1) (2025-07-02)


### Miscellaneous

* release v0.36.1 ([18153df](https://github.com/instill-ai/model-backend/commit/18153dfa88d4ed9f2b54b5342820d316804baf54))

## [0.36.0](https://github.com/instill-ai/model-backend/compare/v0.35.7-alpha...v0.36.0) (2025-06-18)


### Bug Fixes

* **service:** fix profile-image path ([#754](https://github.com/instill-ai/model-backend/issues/754)) ([a696e04](https://github.com/instill-ai/model-backend/commit/a696e04a9532bddcd6ca2b8b88e88f9279050515))


### Miscellaneous

* **proto:** adopt latest `protogen-go` package ([#756](https://github.com/instill-ai/model-backend/issues/756)) ([8460442](https://github.com/instill-ai/model-backend/commit/8460442c171c6a48741d7f833046eea026d284b3))
* release v0.36.0 ([5efe41e](https://github.com/instill-ai/model-backend/commit/5efe41e8907db7fd5d6d5c318b3b4ecaa779dc0d))
* release v0.36.0 ([d9c95d7](https://github.com/instill-ai/model-backend/commit/d9c95d7a2dce9e51c175c204068eb0ab6bebf7d7))


### Tests

* **integration:** remove deprecated API prefix ([#752](https://github.com/instill-ai/model-backend/issues/752)) ([56575a7](https://github.com/instill-ai/model-backend/commit/56575a78ce96e1f669ce95de571caaef31b8e658))

## [0.35.7-alpha](https://github.com/instill-ai/model-backend/compare/v0.35.6-alpha...v0.35.7-alpha) (2025-06-03)


### Miscellaneous

* **domain:** update production domain ([#750](https://github.com/instill-ai/model-backend/issues/750)) ([ab72b40](https://github.com/instill-ai/model-backend/commit/ab72b402ddccbde18b3786ae5ec63d25c1b3287a))

## [0.35.6-alpha](https://github.com/instill-ai/model-backend/compare/v0.35.5-alpha...v0.35.6-alpha) (2025-05-27)


### Miscellaneous

* **ray:** update min_replica value ([#748](https://github.com/instill-ai/model-backend/issues/748)) ([a323a1e](https://github.com/instill-ai/model-backend/commit/a323a1e7c5d4f50bde3dd585176891612e1e9a98))

## [0.35.5-alpha](https://github.com/instill-ai/model-backend/compare/v0.35.4-alpha...v0.35.5-alpha) (2025-05-21)


### Bug Fixes

* **ray:** fix the wrong proto service ([#746](https://github.com/instill-ai/model-backend/issues/746)) ([1d80c4b](https://github.com/instill-ai/model-backend/commit/1d80c4bcf2e5f30642ad48e0122179f6b600d602))

## [0.35.4-alpha](https://github.com/instill-ai/model-backend/compare/v0.35.3-alpha...v0.35.4-alpha) (2025-05-20)


### Miscellaneous Chores

* release v0.35.4-alpha ([81b9089](https://github.com/instill-ai/model-backend/commit/81b9089521c69658c4103b375208123220e98da0))

## [0.35.3-alpha](https://github.com/instill-ai/model-backend/compare/v0.35.2-alpha...v0.35.3-alpha) (2025-04-16)


### Miscellaneous Chores

* release v0.35.3-alpha ([19701b2](https://github.com/instill-ai/model-backend/commit/19701b237b1995df4e95023eb95ab5e6e326eac5))

## [0.35.2-alpha](https://github.com/instill-ai/model-backend/compare/v0.35.1-alpha...v0.35.2-alpha) (2025-04-01)


### Bug Fixes

* **ray:** fix potential loss of deployment config ([#736](https://github.com/instill-ai/model-backend/issues/736)) ([6208b14](https://github.com/instill-ai/model-backend/commit/6208b144b1c045296fa3d9319141f42798e8b90a))

## [0.35.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.35.0-alpha...v0.35.1-alpha) (2025-03-28)


### Miscellaneous Chores

* release v0.35.1-alpha ([31933b9](https://github.com/instill-ai/model-backend/commit/31933b927eebbd1c2e54b4b870ff59800e23c87f))

## [0.35.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.34.0-alpha...v0.35.0-alpha) (2025-02-24)


### Features

* **minio:** add service name and version to MinIO requests ([#728](https://github.com/instill-ai/model-backend/issues/728)) ([5c0d7a3](https://github.com/instill-ai/model-backend/commit/5c0d7a3278c7b7421d23d2fa4bcc8300d54ed4c2))
* **minio:** log MinIO actions with requester ([#724](https://github.com/instill-ai/model-backend/issues/724)) ([f784e0c](https://github.com/instill-ai/model-backend/commit/f784e0c0dad57b4a6468b581632af532030cfaca))
* **x:** update MinIO package to delegate audit logs ([#726](https://github.com/instill-ai/model-backend/issues/726)) ([24b0846](https://github.com/instill-ai/model-backend/commit/24b0846e8aabcc20d2bf5fed1a3132ea066c99e5))


### Bug Fixes

* **ci:** remove docker images only when they exist ([#725](https://github.com/instill-ai/model-backend/issues/725)) ([b6fc25d](https://github.com/instill-ai/model-backend/commit/b6fc25d9c041a7f5ed6683d05c822ff7d73dec3f))
* **mod:** update golang.org/x/net module to fix vulnerability issue ([19f7527](https://github.com/instill-ai/model-backend/commit/19f7527add36e7ee4d2463619b6fcfa4d78ad22a))
* **ray:** fix "GPU" tag deployment ([29f83af](https://github.com/instill-ai/model-backend/commit/29f83af44bb091631dcee9c68117a5103ef2a505))

## [0.34.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.33.1-alpha...v0.34.0-alpha) (2024-12-16)


### Features

* **handler:** add title for region hardware ([#718](https://github.com/instill-ai/model-backend/issues/718)) ([c71bb75](https://github.com/instill-ai/model-backend/commit/c71bb75f7869498c62c2d6a93399f6d6bb633c28))
* **ray:** support multi-gpu suffix ([#714](https://github.com/instill-ai/model-backend/issues/714)) ([bfc442d](https://github.com/instill-ai/model-backend/commit/bfc442d4215384843c4b771e2d8210c439ef448c))
* **ray:** utilize redis as deployment config cache ([#722](https://github.com/instill-ai/model-backend/issues/722)) ([da20daf](https://github.com/instill-ai/model-backend/commit/da20daf573a3d3f399d4ace7a35d0515aba883be))


### Bug Fixes

* **ray:** fix missing deployment configs ([#717](https://github.com/instill-ai/model-backend/issues/717)) ([9d20344](https://github.com/instill-ai/model-backend/commit/9d20344a14912d1de40ffe1cee651727aa006b90))

## [0.33.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.33.0-alpha...v0.33.1-alpha) (2024-12-03)


### Miscellaneous Chores

* **release:** release v0.33.1-alpha ([7ad830f](https://github.com/instill-ai/model-backend/commit/7ad830f97d7c24f8841c5b530b0115f948341e12))

## [0.33.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.32.0-alpha...v0.33.0-alpha) (2024-11-05)


### Features

* **minio:** import updated minio package and add tag on file upload ([#704](https://github.com/instill-ai/model-backend/issues/704)) ([7d03354](https://github.com/instill-ai/model-backend/commit/7d03354b97f1841593f3dee8e21bec4e23d3c829))
* **run:** write data for usage by model trigger chart record api ([#705](https://github.com/instill-ai/model-backend/issues/705)) ([fababc9](https://github.com/instill-ai/model-backend/commit/fababc976e16dbe43118489d385b6db5b7e3bc63))


### Bug Fixes

* **migration:** migrate unsupported task to custom ([#709](https://github.com/instill-ai/model-backend/issues/709)) ([f44afd6](https://github.com/instill-ai/model-backend/commit/f44afd65f0550adc86e7a37aa7f73edca7ab5a04))
* **run:** add metadata retention handler ([#707](https://github.com/instill-ai/model-backend/issues/707)) ([43a6a61](https://github.com/instill-ai/model-backend/commit/43a6a6171813a2afde5691ab0115ce166ab78611))
* **run:** add namespace id in response ([#708](https://github.com/instill-ai/model-backend/issues/708)) ([2940b1c](https://github.com/instill-ai/model-backend/commit/2940b1c295aeaedbea40a5f5ff72376c77d4a2e8))

## [0.32.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.31.0-alpha...v0.32.0-alpha) (2024-10-23)


### Features

* **run:** add requester id in list model run response ([#701](https://github.com/instill-ai/model-backend/issues/701)) ([8c22eec](https://github.com/instill-ai/model-backend/commit/8c22eecd636d6654623ac264b24c4bf98c7f0a09))

## [0.31.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.30.2-alpha...v0.31.0-alpha) (2024-10-22)


### Features

* **run:** run logging data list by requester API ([#695](https://github.com/instill-ai/model-backend/issues/695)) ([b118026](https://github.com/instill-ai/model-backend/commit/b118026c963f4279dacfdcde90f78b116dd91454))
* **service:** standardize the tag naming convention ([#700](https://github.com/instill-ai/model-backend/issues/700)) ([e08be0d](https://github.com/instill-ai/model-backend/commit/e08be0d6a9432818b24ab3126e2e0a716e18002d))


### Bug Fixes

* **ray:** fix upscale deployment failure ([#698](https://github.com/instill-ai/model-backend/issues/698)) ([96563c9](https://github.com/instill-ai/model-backend/commit/96563c95ac8325b7ffe25479868eb48a49464375))
* **run:** fix run logging update function ([#699](https://github.com/instill-ai/model-backend/issues/699)) ([a753fe4](https://github.com/instill-ai/model-backend/commit/a753fe4ac670b048b43de9e430add506ed555333))
* **run:** not return minio error in list model run ([#696](https://github.com/instill-ai/model-backend/issues/696)) ([cbeea05](https://github.com/instill-ai/model-backend/commit/cbeea05698710b55022169f625bfdc6017a9f793))

## [0.30.2-alpha](https://github.com/instill-ai/model-backend/compare/v0.30.1-alpha...v0.30.2-alpha) (2024-10-08)


### Bug Fixes

* **minio:** fix nil logger causing panic ([ef9645e](https://github.com/instill-ai/model-backend/commit/ef9645e11d177df49fc8f71392d994e8b072c93b))
* **test:** switch to minimock and add tests ([#687](https://github.com/instill-ai/model-backend/issues/687)) ([141da9d](https://github.com/instill-ai/model-backend/commit/141da9de40c3600cb58e379ae61d9c92221e19fd))

## [0.30.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.30.0-alpha...v0.30.1-alpha) (2024-09-13)


### Bug Fixes

* **run:** fix Runner missing issue ([#689](https://github.com/instill-ai/model-backend/issues/689)) ([22f2aa0](https://github.com/instill-ai/model-backend/commit/22f2aa02cbef910a079d87a269eee8b32e6c0dda))

## [0.30.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.29.0-alpha...v0.30.0-alpha) (2024-09-10)


### Features

* **task:** support new AI Tasks spec ([#655](https://github.com/instill-ai/model-backend/issues/655)) ([e34d4e1](https://github.com/instill-ai/model-backend/commit/e34d4e19c24e443f6488b95eeeb02470a01d961c))

## [0.29.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.28.0-alpha...v0.29.0-alpha) (2024-08-29)


### Features

* **minio:** added minio connection ([#651](https://github.com/instill-ai/model-backend/issues/651)) ([d17770f](https://github.com/instill-ai/model-backend/commit/d17770f56df19dc3e3999bc11383db321b67cd28))
* **minio:** added minio upload file bytes function ([#656](https://github.com/instill-ai/model-backend/issues/656)) ([1f7b403](https://github.com/instill-ai/model-backend/commit/1f7b403b4e9b872e28f3cabe5b46ac76d420e567))
* **minio:** upgraded to minio v7 and return url and file size ([#654](https://github.com/instill-ai/model-backend/issues/654)) ([4c34bf4](https://github.com/instill-ai/model-backend/commit/4c34bf4e930bba25b5ecc0faea5b7fb7d12fae4d))
* **model:** support model statistics in model message ([#660](https://github.com/instill-ai/model-backend/issues/660)) ([b460778](https://github.com/instill-ai/model-backend/commit/b4607783b63b267d6413ed5ca0b2b9c5f8445f86))
* **model:** support num_of_runs in list filter ([#661](https://github.com/instill-ai/model-backend/issues/661)) ([b161512](https://github.com/instill-ai/model-backend/commit/b161512447ae0fa227ff990d1becac525d22d542))
* **ray:** add separate state for scaling ([#659](https://github.com/instill-ai/model-backend/issues/659)) ([f519c82](https://github.com/instill-ai/model-backend/commit/f519c82d3d501a0ba92eaae3fd51dd3763493473))
* **ray:** support triggering upscale ([#665](https://github.com/instill-ai/model-backend/issues/665)) ([441af13](https://github.com/instill-ai/model-backend/commit/441af13d31be2ba399246ac8fc2d07fa6c585e8a))
* **redis:** replace I/O data retrieval from redis with minio ([#667](https://github.com/instill-ai/model-backend/issues/667)) ([fc46ef8](https://github.com/instill-ai/model-backend/commit/fc46ef8fcf813db55877d1f149fdeb5aff69c32a))
* **run:** fix run logging owner view permission ([#668](https://github.com/instill-ai/model-backend/issues/668)) ([5c07c9a](https://github.com/instill-ai/model-backend/commit/5c07c9a7b3df4ad713a524ed1b02f80e03bfd5b3))


### Bug Fixes

* **ray:** fix upscale request timeout ([#675](https://github.com/instill-ai/model-backend/issues/675)) ([d24eea5](https://github.com/instill-ai/model-backend/commit/d24eea5f9f1b4903ce9b452bdee8a9ba3046cce2))
* **redis:** fix input value overwrite ([#658](https://github.com/instill-ai/model-backend/issues/658)) ([a383380](https://github.com/instill-ai/model-backend/commit/a38338001f4f5fa10917635dd8ad2e14e22af1cc))
* **workflow:** fix duplicate key when activity retry ([#669](https://github.com/instill-ai/model-backend/issues/669)) ([97d9cd2](https://github.com/instill-ai/model-backend/commit/97d9cd247d0e50ed728d11068f555848d80980b8))
* **workflow:** fix state check ([#673](https://github.com/instill-ai/model-backend/issues/673)) ([0514cae](https://github.com/instill-ai/model-backend/commit/0514caec7b223cf1b69bc21934cebaec16afa394))

## [0.28.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.27.0-alpha...v0.28.0-alpha) (2024-07-31)


### Features

* **mgmt:** use explicit `user_id` and `organization_id` in mgmt request ([#638](https://github.com/instill-ai/model-backend/issues/638)) ([4af37e6](https://github.com/instill-ai/model-backend/commit/4af37e695954001da9d2b6da358f990c90bf884f))
* **model:** implement namespace endpoints ([#639](https://github.com/instill-ai/model-backend/issues/639)) ([8c9acac](https://github.com/instill-ai/model-backend/commit/8c9acac740258cbc908f499ebda78680be664541))
* **pkg:** implement fuzze search for namespace and model ID ([#642](https://github.com/instill-ai/model-backend/issues/642)) ([30b11a8](https://github.com/instill-ai/model-backend/commit/30b11a8cff071e21b53a13f7105e900c88a8b605))
* **tag:** allow custom tags for model ([#643](https://github.com/instill-ai/model-backend/issues/643)) ([9c49db6](https://github.com/instill-ai/model-backend/commit/9c49db63e5620691781dd5d3860b6de34e1fec0f))
* **version:** embed version names in model struct ([#645](https://github.com/instill-ai/model-backend/issues/645)) ([88f107b](https://github.com/instill-ai/model-backend/commit/88f107ba31458a1dfa69274080730f27ed4e25a0))


### Bug Fixes

* **repository:** fix missing tag query ([#644](https://github.com/instill-ai/model-backend/issues/644)) ([f1f3229](https://github.com/instill-ai/model-backend/commit/f1f3229557c7b44f9deaba5e6c916d7ac7d6480d))
* **transpiler:** fix expression ([#649](https://github.com/instill-ai/model-backend/issues/649)) ([4680c92](https://github.com/instill-ai/model-backend/commit/4680c92c8b67faae1cc461a81e5f8527738803cf))
* **worker:** added tests for trigger model activity ([#630](https://github.com/instill-ai/model-backend/issues/630)) ([abe562e](https://github.com/instill-ai/model-backend/commit/abe562e855b6c68271340aef00265e13aebcf665))

## [0.27.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.26.0-alpha...v0.27.0-alpha) (2024-07-16)


### Features

* **credit:** INS-5058 added check and collect for model trigger ([#619](https://github.com/instill-ai/model-backend/issues/619)) ([5d55b5b](https://github.com/instill-ai/model-backend/commit/5d55b5b1a98d7719701fc9b611cac2f1509f8420))
* **credit:** refactored check and collect for model trigger ([#629](https://github.com/instill-ai/model-backend/issues/629)) ([0a14f9e](https://github.com/instill-ai/model-backend/commit/0a14f9e042ba0fd5e656363c7b8cb381385c3422))
* **ray:** sync deployment configs when starting up ([#631](https://github.com/instill-ai/model-backend/issues/631)) ([4a2fc8a](https://github.com/instill-ai/model-backend/commit/4a2fc8ad682872a63ed98575f8b04639388958b7))
* **service:** check trigger permissions when impersonating an org ([#635](https://github.com/instill-ai/model-backend/issues/635)) ([427fc45](https://github.com/instill-ai/model-backend/commit/427fc45cc46493e5ea8f7f7ef76389803c20e5e2))


### Bug Fixes

* **worker:** unit test for TriggerModelActivity ([#624](https://github.com/instill-ai/model-backend/issues/624)) ([dec7a60](https://github.com/instill-ai/model-backend/commit/dec7a6052d366a04de2b22f2213d320f09d342c9))

## [0.26.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.25.1-alpha...v0.26.0-alpha) (2024-07-02)


### Features

* **model:** support model version deletion ([#616](https://github.com/instill-ai/model-backend/issues/616)) ([2dca40b](https://github.com/instill-ai/model-backend/commit/2dca40be24409981f95f5f0b1686bd8d2d5771e1))
* **repository:** support case-insensitive search models ([#621](https://github.com/instill-ai/model-backend/issues/621)) ([26c76b2](https://github.com/instill-ai/model-backend/commit/26c76b280d23265cf975d4d1509e7823ea1defa5))


### Bug Fixes

* **redis:** fix misconfigured ttl ([f5da795](https://github.com/instill-ai/model-backend/commit/f5da7958146ea79627e710ee25fe6fcdc2f22abd))
* **worker:** fix mishandled workflow not found ([0189dd8](https://github.com/instill-ai/model-backend/commit/0189dd81715574d6dd985a974fb007cf90102518))

## [0.25.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.25.0-alpha...v0.25.1-alpha) (2024-06-20)


### Bug Fixes

* **schema:** use camelCase for schema fields ([5629b6a](https://github.com/instill-ai/model-backend/commit/5629b6a687e813ab7fd9420e9c96103b08855f9a))

## [0.25.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.24.0-alpha...v0.25.0-alpha) (2024-06-18)


### Features

* **endpoints:** use camelCase for `filter` query string ([#603](https://github.com/instill-ai/model-backend/issues/603)) ([23955e9](https://github.com/instill-ai/model-backend/commit/23955e9a3f3cbb5cdc0fec67091c1275eceed07f))
* **handler:** use camelCase for HTTP body ([#599](https://github.com/instill-ai/model-backend/issues/599)) ([70f6d9a](https://github.com/instill-ai/model-backend/commit/70f6d9ac629ebb5dbb8ffcc5685731ef5c1609c0))
* **model:** support model tag ([#600](https://github.com/instill-ai/model-backend/issues/600)) ([ef87bc9](https://github.com/instill-ai/model-backend/commit/ef87bc9a36a10546559f3590668c49b7e94fc3c5))

## [0.24.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.23.0-alpha...v0.24.0-alpha) (2024-06-06)


### ⚠ BREAKING CHANGES

* **model:** adopt containerized model serving ([#542](https://github.com/instill-ai/model-backend/issues/542))

### Features

* **handler:** implement get latest operation ([#589](https://github.com/instill-ai/model-backend/issues/589)) ([33d2395](https://github.com/instill-ai/model-backend/commit/33d2395f8b89e40f41a5d85adb76be83b590b47b))
* **handler:** support listing available regions for model deployment ([#561](https://github.com/instill-ai/model-backend/issues/561)) ([52c2172](https://github.com/instill-ai/model-backend/commit/52c217272c05e7e80f807bc624008fc48b58e4c7))
* **handler:** support model profile image ([#566](https://github.com/instill-ai/model-backend/issues/566)) ([0c8dbba](https://github.com/instill-ai/model-backend/commit/0c8dbba5a2c51ddf7c87eafc05d916a852e53b13))
* **model:** add permission field in model object ([#576](https://github.com/instill-ai/model-backend/issues/576)) ([2d36a58](https://github.com/instill-ai/model-backend/commit/2d36a584cd37d76b366cf5afcb8762dacdea8200))
* **model:** add task schema in model struct ([#578](https://github.com/instill-ai/model-backend/issues/578)) ([647069d](https://github.com/instill-ai/model-backend/commit/647069d160d9b1bd57281070db6d147b234f37a3))
* **model:** adopt containerized model serving ([#542](https://github.com/instill-ai/model-backend/issues/542)) ([3c80f39](https://github.com/instill-ai/model-backend/commit/3c80f39211c7e0eed76f5e02a310a768496e3d30))
* **model:** embed sample input/output in model proto message ([#558](https://github.com/instill-ai/model-backend/issues/558)) ([5fba538](https://github.com/instill-ai/model-backend/commit/5fba538ab650c107299c0af31354a8f40a02790c))
* **model:** support latest model version trigger ([#580](https://github.com/instill-ai/model-backend/issues/580)) ([47cb36c](https://github.com/instill-ai/model-backend/commit/47cb36c2b877a775ace8356a33e7dc240e1c6b61))
* **model:** support resource spec in model definition ([#557](https://github.com/instill-ai/model-backend/issues/557)) ([fee6e4b](https://github.com/instill-ai/model-backend/commit/fee6e4ba51b5debaf70080ae6afe8233efda1128))
* **model:** support search/filter with list endpoints ([#559](https://github.com/instill-ai/model-backend/issues/559)) ([7b17393](https://github.com/instill-ai/model-backend/commit/7b173938917832c8b1e186c49c35d7d0d15573bd))
* **model:** support watch latest model and `order_by` for list endpoints ([#586](https://github.com/instill-ai/model-backend/issues/586)) ([1a5e48c](https://github.com/instill-ai/model-backend/commit/1a5e48cbb7422e4775166354b510a78fd7ce122c))
* **prediction:** implement sync/async prediction records ([#555](https://github.com/instill-ai/model-backend/issues/555)) ([8d58eda](https://github.com/instill-ai/model-backend/commit/8d58edad0c28c9ee2562efda791f345cee9b61a0))
* **ray:** support containerized model deployment ([#529](https://github.com/instill-ai/model-backend/issues/529)) ([4dcab05](https://github.com/instill-ai/model-backend/commit/4dcab059f1be5ad14242982b19c5cbfd1d0fb822))
* **ray:** support custom accelerator type ([#547](https://github.com/instill-ai/model-backend/issues/547)) ([f0cc0d7](https://github.com/instill-ai/model-backend/commit/f0cc0d761097834618b03033e295429b2f1b41e3))


### Bug Fixes

* **acl:** fix wrong type name ([#560](https://github.com/instill-ai/model-backend/issues/560)) ([89d09a5](https://github.com/instill-ai/model-backend/commit/89d09a57993f50365515d0511c9c0e480992094f))
* **dockerfile:** update deploy config yaml path ([#590](https://github.com/instill-ai/model-backend/issues/590)) ([ee369e0](https://github.com/instill-ai/model-backend/commit/ee369e0a759014a2728c3b271bbc4f63cda1af59))
* **model:** fix missing package in test models ([#552](https://github.com/instill-ai/model-backend/issues/552)) ([a28a21b](https://github.com/instill-ai/model-backend/commit/a28a21b01fecb863bf2720baf2f2e01a344fe808))
* **ray:** check CDI availability for model container ([#538](https://github.com/instill-ai/model-backend/issues/538)) ([28bad42](https://github.com/instill-ai/model-backend/commit/28bad42948b4de2859e7856735d2ca58b194eff7))
* **server:** add missing message size option ([#597](https://github.com/instill-ai/model-backend/issues/597)) ([d0a0aac](https://github.com/instill-ai/model-backend/commit/d0a0aac8fcbb9d80152477666e5474843ba074ba))
* **service:** fix list model version pagination ([#569](https://github.com/instill-ai/model-backend/issues/569)) ([d8fb04a](https://github.com/instill-ai/model-backend/commit/d8fb04ae7a8e5e206a010c056992c01201e02cc7))
* **service:** fix list model version return list size ([#556](https://github.com/instill-ai/model-backend/issues/556)) ([9b69f9c](https://github.com/instill-ai/model-backend/commit/9b69f9c29381d2777da64b0e21a117c3a5113724))

## [0.23.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.22.0-alpha...v0.23.0-alpha) (2024-03-09)


### Features

* **pkg:** use mgmtPB.Owner to embed the owner information ([#523](https://github.com/instill-ai/model-backend/issues/523)) ([37d5708](https://github.com/instill-ai/model-backend/commit/37d57087ab93570399b799ddb6264c4da18f5025))


### Bug Fixes

* **handler,ray:** fix reconciliation model status and namespace ([#525](https://github.com/instill-ai/model-backend/issues/525)) ([62a30b6](https://github.com/instill-ai/model-backend/commit/62a30b64509d5e22820a1480a1c0fa8019e2372a))
* **redis:** delete redis key when errored ([#526](https://github.com/instill-ai/model-backend/issues/526)) ([bb4e18d](https://github.com/instill-ai/model-backend/commit/bb4e18d35e1564282cc5e6f64f3630003a74bc2a))

## [0.22.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.21.3-alpha...v0.22.0-alpha) (2024-02-20)


### ⚠ BREAKING CHANGES

* **triton:** deprecate triton inference server ([#512](https://github.com/instill-ai/model-backend/issues/512))

### Features

* **acl,org:** adopt ACL and add organization endpoints ([#504](https://github.com/instill-ai/model-backend/issues/504)) ([13a1650](https://github.com/instill-ai/model-backend/commit/13a165031544206ed6c6b6b9bb1ac19fc22e6749))


### Bug Fixes

* **cmd,pkg:** refactor codebase to align with `golanci-linter` checks ([#506](https://github.com/instill-ai/model-backend/issues/506)) ([b213812](https://github.com/instill-ai/model-backend/commit/b213812b7e4da8d00fd45261d35cf9ab6a59eafc))
* **handler:** fix multipart request ([352a4ae](https://github.com/instill-ai/model-backend/commit/352a4ae857088c446afb7213b550630d366d3d44))
* **pkg:** fix isError and set maxBatchSize to 0 ([2adfe5b](https://github.com/instill-ai/model-backend/commit/2adfe5bdf185b0e32a39184746bbebf66658af7b))
* **pkg:** fix org model namespace ([#510](https://github.com/instill-ai/model-backend/issues/510)) ([f4be09c](https://github.com/instill-ai/model-backend/commit/f4be09ccb5a1d1d8c122cde61016507294858dfc))
* **service:** fix workflow retry when deleting ([adcbde5](https://github.com/instill-ai/model-backend/commit/adcbde5047d119429981cceaed7f24483e5aa516))
* **service:** remove org subscription check ([76cd66f](https://github.com/instill-ai/model-backend/commit/76cd66feab876ea08ad375298c7657bae2fdca29))
* **usage:** add missing org usage collection ([239d3f4](https://github.com/instill-ai/model-backend/commit/239d3f43ffe0ad8ee7bd456ab33df4b44b99be46))
* **worker:** fix temporal cloud namespace init ([#513](https://github.com/instill-ai/model-backend/issues/513)) ([17c5d68](https://github.com/instill-ai/model-backend/commit/17c5d68a48d9bdb5497e72638ad41aea07eeef16))


### Code Refactoring

* **triton:** deprecate triton inference server ([#512](https://github.com/instill-ai/model-backend/issues/512)) ([f8a277d](https://github.com/instill-ai/model-backend/commit/f8a277d2dc96033672a799f81dd0b09cc4530f30))

## [0.21.3-alpha](https://github.com/instill-ai/model-backend/compare/v0.21.2-alpha...v0.21.3-alpha) (2024-01-30)


### Bug Fixes

* **model:** fix indexing error in text2img and img2img postprocessing ([#501](https://github.com/instill-ai/model-backend/issues/501)) ([0ba505b](https://github.com/instill-ai/model-backend/commit/0ba505bb9c4236590c6669e2e491ef8875eff500))
* **model:** fix missing field in ray while serving img2img task ([#496](https://github.com/instill-ai/model-backend/issues/496)) ([f572f18](https://github.com/instill-ai/model-backend/commit/f572f18f2f35e7330b022cbc4da68f564e1661a5))
* **payload:** fix wrong form data key ([#503](https://github.com/instill-ai/model-backend/issues/503)) ([4d69e5e](https://github.com/instill-ai/model-backend/commit/4d69e5e0322f50b76408ac6d0df6925067a3bb3a))

## [0.21.2-alpha](https://github.com/instill-ai/model-backend/compare/v0.21.1-alpha...v0.21.2-alpha) (2024-01-25)


### Bug Fixes

* **main:** fix misused return statement ([5cbfc3d](https://github.com/instill-ai/model-backend/commit/5cbfc3d57606bf475893c3468a47d46c865e1ee5))

## [0.21.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.21.0-alpha...v0.21.1-alpha) (2024-01-02)


### Features

* **service:** support basic github pat to avoid rate-limit ([#477](https://github.com/instill-ai/model-backend/issues/477)) ([45931ca](https://github.com/instill-ai/model-backend/commit/45931caa0b136accb9abd42b83c128b20b0fe414))


### Miscellaneous Chores

* **release:** release v0.21.1-alpha ([bd320b0](https://github.com/instill-ai/model-backend/commit/bd320b02e4a05ba0c416fd12b94d752db745ed21))

## [0.21.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.20.0-alpha...v0.21.0-alpha) (2023-12-14)


### Features

* **model:** refactoring AI Tasks for Consistency Across Text and Image Generation ([#461](https://github.com/instill-ai/model-backend/issues/461)) ([e827130](https://github.com/instill-ai/model-backend/commit/e827130b1e05a010bca82f2e5c36135e1ff6a578))
* **redis:** use redis for model state caching ([#472](https://github.com/instill-ai/model-backend/issues/472)) ([3b6b977](https://github.com/instill-ai/model-backend/commit/3b6b977a2b168af5152fdbb8ca453610faefed39))


### Bug Fixes

* **model:** fix grpc message size limit issue ([#474](https://github.com/instill-ai/model-backend/issues/474)) ([1ec7ae1](https://github.com/instill-ai/model-backend/commit/1ec7ae135c8a6b8cdf13ca1b30174eb761772521))

## [0.20.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.19.0-alpha...v0.20.0-alpha) (2023-11-30)


### Features

* **model:** Enhancements for Image Inpu in Text 2 Image Task  ([#457](https://github.com/instill-ai/model-backend/issues/457)) ([eb604a1](https://github.com/instill-ai/model-backend/commit/eb604a13a058258c8a58bfdd9ca5f4aafa2363b7))
* **ray:** use shared python executable ([#455](https://github.com/instill-ai/model-backend/issues/455)) ([db9658b](https://github.com/instill-ai/model-backend/commit/db9658bd2065aab93389a612dacbfce9fd3448af))


### Bug Fixes

* **model:** fix deployment reconciliation ([#459](https://github.com/instill-ai/model-backend/issues/459)) ([bac1961](https://github.com/instill-ai/model-backend/commit/bac196181dc2f7d72c3d922a80340fa36b14e938))
* **ray:** fix model file extension ([#453](https://github.com/instill-ai/model-backend/issues/453)) ([424d632](https://github.com/instill-ai/model-backend/commit/424d63286125f599852a5088163e1980a5ca4a06))

## [0.19.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.18.0-alpha...v0.19.0-alpha) (2023-11-11)


### Features

* **model:** Support New Fields for Multi-Modal Model In Text Generation Task and Refactor Existing Ones ([#448](https://github.com/instill-ai/model-backend/issues/448)) ([49bdf5b](https://github.com/instill-ai/model-backend/commit/49bdf5b2fe2a26e78b6564172c778b2721177cd8))
* **ray:** add `ray serve` as model serving backend ([#445](https://github.com/instill-ai/model-backend/issues/445)) ([a9b4005](https://github.com/instill-ai/model-backend/commit/a9b4005697237e85609d5245469c4cfc14e4bd72))


### Bug Fixes

* **predeploy:** fix predeploy model missing triton models reference ([3f296cd](https://github.com/instill-ai/model-backend/commit/3f296cd2b2271798a5c4a8519691738814ef48f6))
* **ray:** fix model healthcheck causing scaling loop ([#450](https://github.com/instill-ai/model-backend/issues/450)) ([4d8cdbf](https://github.com/instill-ai/model-backend/commit/4d8cdbfb10fddbc1642a52e59bcd46a1388cb85c))
* **ray:** fix unziping ray model ([ca79411](https://github.com/instill-ai/model-backend/commit/ca79411dee9e2d8a4b1cf77ee9c2ec1c0a961e8b))
* **service:** fix fail model deletion in state error ([#449](https://github.com/instill-ai/model-backend/issues/449)) ([91125c0](https://github.com/instill-ai/model-backend/commit/91125c0779fc9fcc4557669ea8106600b67c6556))

## [0.18.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.17.2-alpha...v0.18.0-alpha) (2023-10-26)


### Features

* **model:** Enhancements for Llava Model Support and Model Hub File Movement ([#434](https://github.com/instill-ai/model-backend/issues/434)) ([58cb97c](https://github.com/instill-ai/model-backend/commit/58cb97c005722ccba05370513268ecd60be7b5b4))
* **model:** Support for LLM-like models in TRITON Inference Server ([#432](https://github.com/instill-ai/model-backend/issues/432)) ([590eb0b](https://github.com/instill-ai/model-backend/commit/590eb0b8d19a78ea7d1432bce4b22bc3d0a37609))


### Bug Fixes

* **Dockerfile:** fix Python 3.11 using Debian base image ([#438](https://github.com/instill-ai/model-backend/issues/438)) ([2ace6eb](https://github.com/instill-ai/model-backend/commit/2ace6eb91e233db8ed8e0a8ed86758b743be409a))
* **payload:** fix incorrect conversion between integer types ([#440](https://github.com/instill-ai/model-backend/issues/440)) ([32bffea](https://github.com/instill-ai/model-backend/commit/32bffea38c95025de39d52b6e53c55af4b5b0e3a))

## [0.17.2-alpha](https://github.com/instill-ai/model-backend/compare/v0.17.1-alpha...v0.17.2-alpha) (2023-10-13)


### Bug Fixes

* **model:** fix init model namespace ([77a35b3](https://github.com/instill-ai/model-backend/commit/77a35b3eaecb876641ce952342d12e14d6edf0c7))

## [0.17.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.17.0-alpha...v0.17.1-alpha) (2023-09-30)


### Bug Fixes

* **main:** fix namespace error when deploying model ([#423](https://github.com/instill-ai/model-backend/issues/423)) ([dd5badf](https://github.com/instill-ai/model-backend/commit/dd5badf0d6d95babdde033a9a2a441d715f301b1))

## [0.17.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.16.11-alpha...v0.17.0-alpha) (2023-09-13)


### Miscellaneous Chores

* **release:** release v0.17.0-alpha ([70172a2](https://github.com/instill-ai/model-backend/commit/70172a26290c07f6cf8c6b256ccca1c368186e01))

## [0.16.11-alpha](https://github.com/instill-ai/model-backend/compare/v0.16.10-alpha...v0.16.11-alpha) (2023-08-19)


### Miscellaneous Chores

* **release:** release v0.16.11-alpha ([5aba1ce](https://github.com/instill-ai/model-backend/commit/5aba1ceff438915c3f905c815824e61e2aa449e9))

## [0.16.10-alpha](https://github.com/instill-ai/model-backend/compare/v0.16.9-alpha...v0.16.10-alpha) (2023-08-03)


### Miscellaneous Chores

* **release:** release v0.16.10-alpha ([1cd7990](https://github.com/instill-ai/model-backend/commit/1cd79902bff3bba0ff8582dfa453062c6521dbb9))

## [0.16.9-alpha](https://github.com/instill-ai/model-backend/compare/v0.16.8-alpha...v0.16.9-alpha) (2023-07-20)


### Miscellaneous Chores

* **release:** release v0.16.9-alpha ([485a9fd](https://github.com/instill-ai/model-backend/commit/485a9fd5bb6461bbb907363441fc80fbd3ac77dd))

## [0.16.8-alpha](https://github.com/instill-ai/model-backend/compare/v0.16.7-alpha...v0.16.8-alpha) (2023-07-09)


### Miscellaneous Chores

* **release:** release v0.16.8-alpha ([8251037](https://github.com/instill-ai/model-backend/commit/8251037dff3bfca59b4b1f972d1dfbe27d565bea))

## [0.16.7-alpha](https://github.com/instill-ai/model-backend/compare/v0.16.6-alpha...v0.16.7-alpha) (2023-06-20)


### Miscellaneous Chores

* **release:** release 0.16.7-alpha ([c8ef5c4](https://github.com/instill-ai/model-backend/commit/c8ef5c43fe60b990b7dbd20fc9c1c1ab027137f6))

## [0.16.6-alpha](https://github.com/instill-ai/model-backend/compare/v0.16.5-alpha...v0.16.6-alpha) (2023-06-11)


### Miscellaneous Chores

* **release:** release v0.16.6-alpha ([c1f57a9](https://github.com/instill-ai/model-backend/commit/c1f57a941794d7e6dff1e4d053383247c08ad595))

## [0.16.5-alpha](https://github.com/instill-ai/model-backend/compare/v0.16.4-alpha...v0.16.5-alpha) (2023-06-02)


### Miscellaneous Chores

* **release:** release v0.16.5-alpha ([b8ba368](https://github.com/instill-ai/model-backend/commit/b8ba3685a3b180812c8463efd47cc4ddbe5a08ec))

## [0.16.4-alpha](https://github.com/instill-ai/model-backend/compare/v0.16.3-alpha...v0.16.4-alpha) (2023-05-11)


### Miscellaneous Chores

* **release:** release v0.16.4-alpha ([ab8cf12](https://github.com/instill-ai/model-backend/commit/ab8cf12e0b9a7b211d8e8a12c06d683604db5017))

## [0.16.3-alpha](https://github.com/instill-ai/model-backend/compare/v0.16.2-alpha...v0.16.3-alpha) (2023-05-06)


### Bug Fixes

* create single triton client ([#357](https://github.com/instill-ai/model-backend/issues/357)) ([8dedf5d](https://github.com/instill-ai/model-backend/commit/8dedf5d2c77279a15f906df580a93c46b21cc046))

## [0.16.2-alpha](https://github.com/instill-ai/model-backend/compare/v0.16.1-alpha...v0.16.2-alpha) (2023-04-25)


### Miscellaneous Chores

* **release:** release v0.16.2-alpha ([b735b17](https://github.com/instill-ai/model-backend/commit/b735b170a23249de6798afdc5f326cbba9140385))

## [0.16.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.16.0-alpha...v0.16.1-alpha) (2023-04-24)


### Bug Fixes

* pass the context between package layers ([#345](https://github.com/instill-ai/model-backend/issues/345)) ([e6e7f2f](https://github.com/instill-ai/model-backend/commit/e6e7f2fa256390337593b1a039c93c01a1779a98))


### Miscellaneous Chores

* release 0.16.1-alpha ([1d497e0](https://github.com/instill-ai/model-backend/commit/1d497e01dec349cfc6b711ee05ed53fd7406765a))

## [0.16.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.15.0-alpha...v0.16.0-alpha) (2023-04-15)


### Features

* add model initialization module ([#332](https://github.com/instill-ai/model-backend/issues/332)) ([aa753a5](https://github.com/instill-ai/model-backend/commit/aa753a5eb1b40c0cdee23142328bd9bfc56d85de))

## [0.15.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.14.0-alpha...v0.15.0-alpha) (2023-04-07)


### Features

* **controller:** add model state monitoring with controller ([#323](https://github.com/instill-ai/model-backend/issues/323)) ([4397826](https://github.com/instill-ai/model-backend/commit/43978264209011031d3622b1336e7bbdf237d985))
* remove model instance ([#320](https://github.com/instill-ai/model-backend/issues/320)) ([15e1b62](https://github.com/instill-ai/model-backend/commit/15e1b625e5c2d876c580c9a6906c18b600cd7c7b))
* support model caching ([#317](https://github.com/instill-ai/model-backend/issues/317)) ([d15ffba](https://github.com/instill-ai/model-backend/commit/d15ffba489ad985ecfafdd3038c6a97630da94fc))

## [0.14.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.13.1-alpha...v0.14.0-alpha) (2023-03-26)


### Features

* add private endpoint and gRPC test cases ([#306](https://github.com/instill-ai/model-backend/issues/306)) ([bb3c193](https://github.com/instill-ai/model-backend/commit/bb3c19321305c83407e47a19929db2b3f71ac5b0))


### Bug Fixes

* **config:** use private port for mgmt-backend ([#307](https://github.com/instill-ai/model-backend/issues/307)) ([3264e2b](https://github.com/instill-ai/model-backend/commit/3264e2b5358a393d0027b21e2b56ad55c72dcb6c))
* list models and model instances pagination ([#304](https://github.com/instill-ai/model-backend/issues/304)) ([1f19ed4](https://github.com/instill-ai/model-backend/commit/1f19ed4796dc04610a7496918c6e29bc6afb51e0))

## [0.13.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.13.0-alpha...v0.13.1-alpha) (2023-02-26)


### Bug Fixes

* create a subfolder in model-repository if needed ([#290](https://github.com/instill-ai/model-backend/issues/290)) ([7f8d78b](https://github.com/instill-ai/model-backend/commit/7f8d78b89dfa57ba0a065568b62b3bea3e0cae12))
* fix creating subfolder ([105a11a](https://github.com/instill-ai/model-backend/commit/105a11a956fc6b0b5150587d5ec3bba08b54b1b9))
* fix subfolder creation ([#292](https://github.com/instill-ai/model-backend/issues/292)) ([0b6ec3f](https://github.com/instill-ai/model-backend/commit/0b6ec3fae13f44e68fea7644880a3491eea4c708))
* fix variable name ([#293](https://github.com/instill-ai/model-backend/issues/293)) ([a7995dd](https://github.com/instill-ai/model-backend/commit/a7995dd0d35181b96df7371027ab10609e45b6af))

## [0.13.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.12.1-alpha...v0.13.0-alpha) (2023-02-23)


### Features

* add support for text generation tasks ([#252](https://github.com/instill-ai/model-backend/issues/252)) ([767ec45](https://github.com/instill-ai/model-backend/commit/767ec456c0ff3416343d8f2fb19e621b872806c6))


### Bug Fixes

* keep format for empty inference output ([#258](https://github.com/instill-ai/model-backend/issues/258)) ([e2a2e48](https://github.com/instill-ai/model-backend/commit/e2a2e48e6049026fb072436820863f14aa424b1c))

## [0.12.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.12.0-alpha...v0.12.1-alpha) (2023-02-12)


### Bug Fixes

* fix keypoint model payload parser ([#249](https://github.com/instill-ai/model-backend/issues/249)) ([461d54a](https://github.com/instill-ai/model-backend/commit/461d54a99463cdcf58e6567f5eb41e76515acd9d))

## [0.12.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.11.1-alpha...v0.12.0-alpha) (2023-02-10)


### Features

* add text to image task ([#239](https://github.com/instill-ai/model-backend/issues/239)) ([421eb1a](https://github.com/instill-ai/model-backend/commit/421eb1aa203fc297e4df22b22a72e419329b5869))


### Bug Fixes

* fix usage client nil issue when mgmt-backend not ready ([#241](https://github.com/instill-ai/model-backend/issues/241)) ([4290159](https://github.com/instill-ai/model-backend/commit/429015957de074a0ea2e68a7cb2423a61829c5f1))

## [0.11.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.11.0-alpha...v0.11.1-alpha) (2023-01-20)


### Bug Fixes

* fix list long-run operation error ([#220](https://github.com/instill-ai/model-backend/issues/220)) ([472696d](https://github.com/instill-ai/model-backend/commit/472696dd974e996e247c66750033f0b724668bfc))

## [0.11.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.10.0-alpha...v0.11.0-alpha) (2023-01-14)


### Miscellaneous Chores

* release 0.11.0-alpha ([d592acb](https://github.com/instill-ai/model-backend/commit/d592acbd23de661cb5f695ca6e8f37195452abf9))

## [0.10.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.9.1-alpha...v0.10.0-alpha) (2022-12-23)


### Features

* support async deploy and undeploy model instance ([#192](https://github.com/instill-ai/model-backend/issues/192)) ([ed36dc7](https://github.com/instill-ai/model-backend/commit/ed36dc77df2819be822e57bb7020e2bd06cb2edc))
* support semantic segmentation ([#203](https://github.com/instill-ai/model-backend/issues/203)) ([f22262c](https://github.com/instill-ai/model-backend/commit/f22262cf4da1c64f9e45244a76baf2680ae4dd5d))


### Bug Fixes

* model instance state update to unspecified state ([#206](https://github.com/instill-ai/model-backend/issues/206)) ([14c87d5](https://github.com/instill-ai/model-backend/commit/14c87d5afc3a7a1ad957ff1a05908b14c9902d0c))
* panic error with nil object ([#208](https://github.com/instill-ai/model-backend/issues/208)) ([a342113](https://github.com/instill-ai/model-backend/commit/a342113ae119646e7de775cb2d8d5f3e7e082f58))

## [0.9.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.9.0-alpha...v0.9.1-alpha) (2022-11-28)


### Bug Fixes

* HuggingFace batching bug in preprocess model ([b1582e8](https://github.com/instill-ai/model-backend/commit/b1582e8f48dd5a88a1ad8f8dcddce13d382b9e86))

## [0.9.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.8.1-alpha...v0.9.0-alpha) (2022-10-19)


### Features

* support instance segmentation task ([#183](https://github.com/instill-ai/model-backend/issues/183)) ([d28cfdc](https://github.com/instill-ai/model-backend/commit/d28cfdc50ecca72571c2bfd0cdf53dd2bab6567c))


### Bug Fixes

* allow updating emtpy description for a model ([#177](https://github.com/instill-ai/model-backend/issues/177)) ([100ec84](https://github.com/instill-ai/model-backend/commit/100ec84eed90ca7d3ec7fd04117d0ecc1e40cd22))

## [0.8.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.8.0-alpha...v0.8.1-alpha) (2022-09-19)


### Bug Fixes

* update description for GitHub model from user input ([#173](https://github.com/instill-ai/model-backend/issues/173)) ([821dab3](https://github.com/instill-ai/model-backend/commit/821dab3768dad53f1c4e49ac786e758643825eb3))

## [0.8.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.7.3-alpha...v0.8.0-alpha) (2022-09-14)


### Features

* add confidence score for ocr output ([#167](https://github.com/instill-ai/model-backend/issues/167)) ([e915452](https://github.com/instill-ai/model-backend/commit/e91545247b6128c84117d17dffc4dede171d2e3f))

## [0.7.3-alpha](https://github.com/instill-ai/model-backend/compare/v0.7.2-alpha...v0.7.3-alpha) (2022-09-07)


### Features

* handle oom ([#163](https://github.com/instill-ai/model-backend/issues/163)) ([4db1c45](https://github.com/instill-ai/model-backend/commit/4db1c45da75e308b85561b8d496d097671289c45))


### Miscellaneous Chores

* release 0.7.3-alpha ([9033c50](https://github.com/instill-ai/model-backend/commit/9033c502eaa36b4885cba4ef1add4f5353c1a5ff))

## [0.7.2-alpha](https://github.com/instill-ai/model-backend/compare/v0.7.1-alpha...v0.7.2-alpha) (2022-08-22)


### Miscellaneous Chores

* release 0.7.2-alpha ([17529d6](https://github.com/instill-ai/model-backend/commit/17529d6fad2124d1ed05acfd59c182ec9b9faec7))

## [0.7.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.7.0-alpha...v0.7.1-alpha) (2022-08-21)


### Bug Fixes

* post process ocr task ([e387154](https://github.com/instill-ai/model-backend/commit/e38715481b18cbcecf25b5fba841ff887909dcc1))

## [0.7.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.6.3-alpha...v0.7.0-alpha) (2022-08-17)


### Features

* add release stage for model definition ([#153](https://github.com/instill-ai/model-backend/issues/153)) ([4e13ba5](https://github.com/instill-ai/model-backend/commit/4e13ba5ff13a407de932d084468dd72cb36fd108))
* support ocr task ([#150](https://github.com/instill-ai/model-backend/issues/150)) ([7766c6f](https://github.com/instill-ai/model-backend/commit/7766c6fd82e2a711333c8131fb8fd82a8f462224))

## [0.6.3-alpha](https://github.com/instill-ai/model-backend/compare/v0.6.2-alpha...v0.6.3-alpha) (2022-07-19)


### Bug Fixes

* fix client stream server recv wrong file length interval ([#143](https://github.com/instill-ai/model-backend/issues/143)) ([0e06f7c](https://github.com/instill-ai/model-backend/commit/0e06f7c32fcde81db61ae40f8f5aa35b51ec7000))
* post process for unspecified task output ([ad88068](https://github.com/instill-ai/model-backend/commit/ad880680abd382e175d60428a2864ca36168341f))
* trigger image with 4 channel ([#141](https://github.com/instill-ai/model-backend/issues/141)) ([7445f5f](https://github.com/instill-ai/model-backend/commit/7445f5fcd4c796aa56c11f47c75244f0acf49411))

## [0.6.2-alpha](https://github.com/instill-ai/model-backend/compare/v0.6.1-alpha...v0.6.2-alpha) (2022-07-12)


### Miscellaneous Chores

* release v0.6.2-alpha ([4365f32](https://github.com/instill-ai/model-backend/commit/4365f32207c4e914a89cb030489e153f12a43cd6))

## [0.6.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.6.0-alpha...v0.6.1-alpha) (2022-07-11)


### Miscellaneous Chores

* release v0.6.1-alpha ([f18dc30](https://github.com/instill-ai/model-backend/commit/f18dc306d29cf8fce57ec55476aa632f1d4a12d0))

## [0.6.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.5.1-alpha...v0.6.0-alpha) (2022-07-06)


### Features

* support creating a HuggingFace model ([#113](https://github.com/instill-ai/model-backend/issues/113)) ([1577d87](https://github.com/instill-ai/model-backend/commit/1577d87b58b0c8674276fe85d5762fc3a30d566c))


### Bug Fixes

* model definition in list model and missing zero in output ([#121](https://github.com/instill-ai/model-backend/issues/121)) ([a90072d](https://github.com/instill-ai/model-backend/commit/a90072d19f24d3df9e31ac447a992abf2ec8e525))

## [0.5.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.5.0-alpha...v0.5.1-alpha) (2022-06-27)


### Miscellaneous Chores

* release v0.5.1-alpha ([895056d](https://github.com/instill-ai/model-backend/commit/895056dcca2a5b155980d8032a32db20271eaa62))

## [0.5.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.4.2-alpha...v0.5.0-alpha) (2022-06-26)


### Features

* add credential definition ([#109](https://github.com/instill-ai/model-backend/issues/109)) ([92d3391](https://github.com/instill-ai/model-backend/commit/92d3391ef69a8df83dff8ce528439345e3238073))
* support artivc ([#102](https://github.com/instill-ai/model-backend/issues/102)) ([b8e21a4](https://github.com/instill-ai/model-backend/commit/b8e21a426445e9e40c8cff559be05fc7b1f724e0))


### Bug Fixes

* bug usage storage ([#103](https://github.com/instill-ai/model-backend/issues/103)) ([975fdc1](https://github.com/instill-ai/model-backend/commit/975fdc1e2ed93f13dc5b56772eda1e9ca59c6a2f))
* fix duration configuration bug ([ee4a310](https://github.com/instill-ai/model-backend/commit/ee4a31083fb08670d9b342c22d79c1710b0e57fe))
* init config before logger ([9d3fb4a](https://github.com/instill-ai/model-backend/commit/9d3fb4a0ba0948819b14a7f420035345df7c4d4e))
* status code when deploy model error ([#111](https://github.com/instill-ai/model-backend/issues/111)) ([31d3f11](https://github.com/instill-ai/model-backend/commit/31d3f11ba04ee59b12521b8e0dd724849a81b94f))
* update model definitions and tasks in usage collection ([#100](https://github.com/instill-ai/model-backend/issues/100)) ([c593087](https://github.com/instill-ai/model-backend/commit/c5930870595c5d280d7db005a711c0cc9bff802c))
* wrong logic when checking user account and service account ([7058db6](https://github.com/instill-ai/model-backend/commit/7058db643bfa9b852164f612c7b2fc5ca65260e8))

### [0.4.2-alpha](https://github.com/instill-ai/model-backend/compare/v0.4.1-alpha...v0.4.2-alpha) (2022-05-31)


### Bug Fixes

* fix config path ([a8cf2c0](https://github.com/instill-ai/model-backend/commit/a8cf2c01e7ec512d93abf24c98d991d75ea4258e))
* regexp zap logger with new protobuf package ([8b9c463](https://github.com/instill-ai/model-backend/commit/8b9c4632c9303db090e910d6ac939ff794f56e31))


### Miscellaneous Chores

* release 0.4.2-alpha ([fc5a14a](https://github.com/instill-ai/model-backend/commit/fc5a14a4e779d92ea5d19eed857d4e1b27683b26))

### [0.4.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.4.0-alpha...v0.4.1-alpha) (2022-05-19)


### Bug Fixes

* add writeonly to description ([f59d98f](https://github.com/instill-ai/model-backend/commit/f59d98f20cbf67168fe8b15cf085f950b858ea9a))
* clone repository and make folder ([ac79386](https://github.com/instill-ai/model-backend/commit/ac793865218e1d1cd7c9d6d6017329b66821b626))
* model configuration response in integration test ([0225c1e](https://github.com/instill-ai/model-backend/commit/0225c1ef7bbb461fb79126232a53ffe2015e5eb0))
* refactor JSON schema ([f24db48](https://github.com/instill-ai/model-backend/commit/f24db48bd2b5fe5c12d12962c63146a7388031c3))

## [0.4.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.3.2-alpha...v0.4.0-alpha) (2022-05-13)


### Features

* create model from GitHub ([#61](https://github.com/instill-ai/model-backend/issues/61)) ([cf763cb](https://github.com/instill-ai/model-backend/commit/cf763cb715caf665bd9aa8dab25f621a81a22aa8))


### Bug Fixes

* refactor model definition and model JSON schema ([#73](https://github.com/instill-ai/model-backend/issues/73)) ([0cce154](https://github.com/instill-ai/model-backend/commit/0cce154f90af85d12fc2b608e468b2122bb63920))

### [0.3.2-alpha](https://github.com/instill-ai/model-backend/compare/v0.3.1-alpha...v0.3.2-alpha) (2022-03-22)


### Miscellaneous Chores

* release 0.3.2-alpha ([9f8cd91](https://github.com/instill-ai/model-backend/commit/9f8cd91a2ac90a193b534cfaaf39a2c03815816c))

### [0.3.1-alpha](https://github.com/instill-ai/model-backend/compare/v0.3.0-alpha...v0.3.1-alpha) (2022-03-21)


### Bug Fixes

* fix unload model issue causing Triton server OOM ([#42](https://github.com/instill-ai/model-backend/issues/42)) ([fb4d1d1](https://github.com/instill-ai/model-backend/commit/fb4d1d13b846659ad57c3e190c793b3e3caacce0))
* update version order when get model version list ([#38](https://github.com/instill-ai/model-backend/issues/38)) ([83c054a](https://github.com/instill-ai/model-backend/commit/83c054abc4ef8aa0e95e2d6d832f5c6946a9bbd9))

## [0.3.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.2.0-alpha...v0.3.0-alpha) (2022-02-24)


### Features

* support url/base64 content prediction ([#34](https://github.com/instill-ai/model-backend/issues/34)) ([a88ddfd](https://github.com/instill-ai/model-backend/commit/a88ddfd5e266b848053e899eb387ec77555305f3))


### Bug Fixes

* correct version when making inference ([#31](https://github.com/instill-ai/model-backend/issues/31)) ([c918e77](https://github.com/instill-ai/model-backend/commit/c918e77b5a573adc39badf25ecac320204e7fbcc))
* update docker compose file for building dev image ([#29](https://github.com/instill-ai/model-backend/issues/29)) ([83cba09](https://github.com/instill-ai/model-backend/commit/83cba09545179d4e60fa57810143adff674e6a09))

## [0.2.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.1.0-alpha...v0.2.0-alpha) (2022-02-19)


### Features

* add gRPC Gateway and GetModel API ([#7](https://github.com/instill-ai/model-backend/issues/7)) ([bff6fc9](https://github.com/instill-ai/model-backend/commit/bff6fc9431528b0c01066adc3c6e75e9183b457b))
* support model name when creating model ([#25](https://github.com/instill-ai/model-backend/issues/25)) ([7d799b7](https://github.com/instill-ai/model-backend/commit/7d799b7e0936099907a5b12128d3df6183b73fd0))


### Bug Fixes

* fix build and go version ([#9](https://github.com/instill-ai/model-backend/issues/9)) ([f8d4346](https://github.com/instill-ai/model-backend/commit/f8d4346332f117ee4a2a54a390fa1cb3af43cbfb))

## [0.1.0-alpha](https://github.com/instill-ai/model-backend/compare/v0.0.0-alpha...v0.1.0-alpha) (2022-02-12)


### Features

* add codebase for model grpc service ([4defa3e](https://github.com/instill-ai/model-backend/commit/4defa3e203b867940309fd29300ec00efb8b076c))


### Bug Fixes

* add link for guideline create Conda environment file ([7ee8e06](https://github.com/instill-ai/model-backend/commit/7ee8e06079463a9d722fca5c350a8b259a09b4a5))
* logic when essemble or not ([ab8e7c1](https://github.com/instill-ai/model-backend/commit/ab8e7c12da66cc66a0b649f8fc6cbe17f88147f6))
* postgres host ([a322165](https://github.com/instill-ai/model-backend/commit/a322165f28a39a29ecf755f3b6fc6ee55cf3bdd3))
* return list of models in list method ([b88ebd7](https://github.com/instill-ai/model-backend/commit/b88ebd7950b52a075ffff488d69d67d2a49aad99))
* update db schema, protobuf generated files and create model, version in upload api ([7573e54](https://github.com/instill-ai/model-backend/commit/7573e5477b9e4613bd265761ada6d0afd1c31303))
* update predict for essemble model ([016f11c](https://github.com/instill-ai/model-backend/commit/016f11c2df9fdc8f399ae3a51fe56d45ab4f4638))
