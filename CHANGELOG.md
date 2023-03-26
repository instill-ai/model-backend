# Changelog

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
