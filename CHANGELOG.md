# Changelog

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
