# Changelog

## [0.17.0](https://github.com/looplab/eventhorizon/compare/v0.16.0...v0.17.0) (2026-06-16)


### Features

* add UnregisterAggregate for runtime type management ([5e76116](https://github.com/looplab/eventhorizon/commit/5e76116da63eb489fd2c1db3e9cf53b28d07641b))
* **lint:** upgrade golangci-lint to 2.10.0 ([c8cae33](https://github.com/looplab/eventhorizon/commit/c8cae33c81e6db5100ece39d7897e020b8f1530b))
* **lint:** upgrate golangci-lint to 2.8.0 ([00990f0](https://github.com/looplab/eventhorizon/commit/00990f0e37b9fbfd78ed260895e0d6d6b52c7192))
* **mongodb_v2:** add WithGlobalPositionStrategy option ([7ba1d0b](https://github.com/looplab/eventhorizon/commit/7ba1d0b4a0afc8e2394aa0e437e652ea365e1784))
* **nix:** add nix develop flake ([780c160](https://github.com/looplab/eventhorizon/commit/780c1608cd19c520104e5f6224003c36e3afe11c))


### Bug Fixes

* **build:** update to Go 1.23 ([96b3294](https://github.com/looplab/eventhorizon/commit/96b3294edec9b15d4260d5f1ec33f8ce6f21dd6f))
* **build:** update to Go 1.23 ([ffc1c68](https://github.com/looplab/eventhorizon/commit/ffc1c68ef3fb4aad0bbaf22928b0d3eb8b268f2f))
* **ci:** update to up/download-artifact v4 ([877bcc0](https://github.com/looplab/eventhorizon/commit/877bcc001419734c1e964b4683233a61394b46ba))
* **ci:** update to up/download-artifact v4 ([202555f](https://github.com/looplab/eventhorizon/commit/202555feebf4c7c3d89f9b23b3661372e2a70110))
* **ci:** use multiple artifacts for code coverage ([b211160](https://github.com/looplab/eventhorizon/commit/b21116000e9e7b6c8d0e40ed2958ca446ca2c44d))
* **ci:** use multiple artifacts for code coverage ([8bd29cd](https://github.com/looplab/eventhorizon/commit/8bd29cda7bb51236acb99972fd36e3e7dd7c3ad3))
* clean up global registries in tests to support -count&gt;1 ([78cb7b8](https://github.com/looplab/eventhorizon/commit/78cb7b81503549ed392bba44219fdbd6e45dec98))
* correct typo "handeled" to "handled" in async middleware test ([5b9d8f5](https://github.com/looplab/eventhorizon/commit/5b9d8f5b6f90a3c201017562185b23f94d9f6442))
* eliminate flaky tests with synctest ([0a9fd82](https://github.com/looplab/eventhorizon/commit/0a9fd8211da290134a6b6545b99e7a63469eac95))
* **eventbus/gcp:** migrate to cloud.google.com/go/pubsub/v2 ([4deb4f5](https://github.com/looplab/eventhorizon/commit/4deb4f5d2753f7838cf55feee8c2c77b95037154))
* **eventbus/gcp:** migrate to cloud.google.com/go/pubsub/v2 ([14708b6](https://github.com/looplab/eventhorizon/commit/14708b69bf33c56cfea80820efcec73aa100eee3))
* **examples:** rename ExampleIntegration to Example_integration ([60e7dd5](https://github.com/looplab/eventhorizon/commit/60e7dd55462dd27d537c81ad96ecfce40244ad72))
* **kafka:** use Kafka 3.9.0 and upgrade kafka-go to v0.4.47 ([a3bfb18](https://github.com/looplab/eventhorizon/commit/a3bfb18c81f8e8d0b9279d567e216ea6fab55eb3))
* **lint:** check unchecked error return values (errcheck) ([9571c93](https://github.com/looplab/eventhorizon/commit/9571c93356bfa70252542ddb867d7b2d38eb1501))
* **lint:** convert remaining for loops to range syntax (intrange) ([6cb07d5](https://github.com/looplab/eventhorizon/commit/6cb07d5974329aac2b1fb4a557c9f165dcb71b24))
* **lint:** migrate golangci-lint config from v1 to v2 ([1c14bb8](https://github.com/looplab/eventhorizon/commit/1c14bb890fd5a1a144032474c7c36771197c76eb))
* **lint:** modernize for loops to use range (intrange) ([447fbf5](https://github.com/looplab/eventhorizon/commit/447fbf53048995f3128ec88a184b028988fbad2b))
* **lint:** remove underscore receiver names (staticcheck ST1006) ([e965df4](https://github.com/looplab/eventhorizon/commit/e965df46560ae0d08cd18c8a1d8580ce17831019))
* **lint:** remove unnecessary loop variable copies (copyloopvar) ([0e3dfb7](https://github.com/looplab/eventhorizon/commit/0e3dfb7d511843a4a4c28ce3374f4e0eb14e93ca))
* **lint:** remove unnecessary whitespace ([00d7dea](https://github.com/looplab/eventhorizon/commit/00d7deaa86fce78bb22541ed546be4208be3cca8))
* **lint:** remove unused code ([cb8c825](https://github.com/looplab/eventhorizon/commit/cb8c8252f84ad0de37acfb68bfdebe9e326a0697))
* **lint:** replace fmt.Errorf/Sprintf with errors.New and string concat ([4834346](https://github.com/looplab/eventhorizon/commit/4834346948b82ca613989ade4e2d6f8e4165a87b))
* **lint:** resolve contextcheck issues ([ab9bbd9](https://github.com/looplab/eventhorizon/commit/ab9bbd9e7ae01feafc9434bb18856e87beb50190))
* **lint:** resolve errorlint issues ([68b47c5](https://github.com/looplab/eventhorizon/commit/68b47c560d26a4324ce981dab9f0cbe29cbff7e6))
* **lint:** resolve gocritic issues ([b3a36fd](https://github.com/looplab/eventhorizon/commit/b3a36fd47edaa5afd9640f88b341e2e58c5dfa41))
* **lint:** resolve gosec and misc warnings ([ee618a3](https://github.com/looplab/eventhorizon/commit/ee618a38f83239d7977662e0d15ff80db14319b5))
* **lint:** resolve nakedret issue in SaveSnapshot ([4804faf](https://github.com/looplab/eventhorizon/commit/4804faf094ee72b89617d37964b315ca7dbc540e))
* **lint:** resolve staticcheck issues ([cf2b901](https://github.com/looplab/eventhorizon/commit/cf2b9016eceff1a2b0042db972b8e43b82ae737f))
* **lint:** resolve unparam issues ([74bae63](https://github.com/looplab/eventhorizon/commit/74bae6326e0811489b31ec7684ac705f12f16953))
* **lint:** use errors.New for static error strings (perfsprint) ([c82166f](https://github.com/looplab/eventhorizon/commit/c82166ffd1ed356d41bdad2eebbd73abf3db5b0b))
* **lint:** use http method constants (usestdlibvars) ([d770380](https://github.com/looplab/eventhorizon/commit/d770380b14754e470ab500b8c8e2725191ee1b86))
* **lint:** use modern Go APIs (modernize) ([c86a14e](https://github.com/looplab/eventhorizon/commit/c86a14ee2fcf0aafa479a807a1938448079b6d6b))
* **lint:** use typed context keys and fix ineffectual assignment ([ebc5145](https://github.com/looplab/eventhorizon/commit/ebc514523b85fcc588badaeb88ddc21265f466f6))
* **makefile:** use Docker Compose v2 ([b3c17d9](https://github.com/looplab/eventhorizon/commit/b3c17d9285061291bccee2a2f95319169c36e6bf))
* **mongodb:** enable single-node replica set for transactions and change streams ([f566469](https://github.com/looplab/eventhorizon/commit/f566469e999579a72b12806702148585c77b17cb))
* **scheduler:** increase timing margin in TestMiddleware_Persisted ([b33b514](https://github.com/looplab/eventhorizon/commit/b33b51429f737f1eacd71bf79cd0d7f3a4aa4bea))
* **scheduler:** use synctest to fix TestMiddleware_Cancel flakiness ([32c66c5](https://github.com/looplab/eventhorizon/commit/32c66c5176165e825492f9cf5b5c6fb6082065e3))
* **tests:** apply synctest to flaky tests across middleware packages ([b3a2cb4](https://github.com/looplab/eventhorizon/commit/b3a2cb442dfb6c89387e6d1ffc99e3d4091b033d))
* **uuid:** remove unnecessary type conversions ([14054b2](https://github.com/looplab/eventhorizon/commit/14054b2ea493331c8fc960ae7d390ee1ecee8b5d))


### Dependencies

* **go:** upgrade 1.17 to 1.25.6 ([edc65a1](https://github.com/looplab/eventhorizon/commit/edc65a15babdb128832374939c2fcf73ff2909b0))


### Documentation

* **mongodb_v2:** add STRATEGY.md documenting global position trade-offs ([a3d2298](https://github.com/looplab/eventhorizon/commit/a3d2298ee86b6af13c5f047c0573bb764e15719c))


### Miscellaneous

* bump Go to 1.25.6 ([f5627e9](https://github.com/looplab/eventhorizon/commit/f5627e95c2a7a6edb2f777aa02f763ca9ec659cc))
* **deps:** bump github.com/nats-io/nats-server/v2 ([7df1b3a](https://github.com/looplab/eventhorizon/commit/7df1b3a07ed9b67e016fc473b196cbd27ba5c015))
* **deps:** bump github.com/nats-io/nats-server/v2 from 2.6.2 to 2.7.4 ([89f5710](https://github.com/looplab/eventhorizon/commit/89f571021a5b40c2cdc6254531c859ef397a3177))
* **deps:** bump github.com/nats-io/nats-server/v2 from 2.7.4 to 2.11.12 ([6cc4453](https://github.com/looplab/eventhorizon/commit/6cc4453f18dd791432fe0e63bfb77c539f8fba7e))
* **deps:** bump golang.org/x/oauth2 ([dbf2afa](https://github.com/looplab/eventhorizon/commit/dbf2afa869d51b36889e037e4940d629a286c1e2))
* **deps:** bump golang.org/x/oauth2 from 0.0.0-20211104180415-d3ed0bb246c8 to 0.27.0 ([335f072](https://github.com/looplab/eventhorizon/commit/335f0726a58b368b3a230d742f1849cc4ab8f0ba))
* **deps:** bump google.golang.org/grpc from 1.40.0 to 1.56.3 ([4bc5eac](https://github.com/looplab/eventhorizon/commit/4bc5eac8c1e37b4846bf48deaaa72f7ff58af7e4))
* **deps:** bump google.golang.org/grpc from 1.40.0 to 1.56.3 ([420046c](https://github.com/looplab/eventhorizon/commit/420046c1813d61a658a4623b662598f9e36e1c1d))
* **deps:** bump google.golang.org/protobuf from 1.27.1 to 1.33.0 ([d49b59b](https://github.com/looplab/eventhorizon/commit/d49b59b9d1ff58f243e039ef4614ab2ff297cc30))
* **deps:** bump google.golang.org/protobuf from 1.27.1 to 1.33.0 ([6e9a9d9](https://github.com/looplab/eventhorizon/commit/6e9a9d9aa3b13de64e4b33ba137e4b02e82bd7dd))
* **deps:** bump gopkg.in/yaml.v3 ([487d6d2](https://github.com/looplab/eventhorizon/commit/487d6d27ea7ee6add8d020d789c018ea4937ba6a))
* **deps:** bump gopkg.in/yaml.v3 from 3.0.0-20200313102051-9f266ea9e77c to 3.0.1 ([fed4316](https://github.com/looplab/eventhorizon/commit/fed4316cc00cfb191f438003e063f505cf4f6129))
* pkg imported more than once ([86050e1](https://github.com/looplab/eventhorizon/commit/86050e1cb716ba403214bfcc44a0a35de705f14a))
* pkg imported more than once ([556fa51](https://github.com/looplab/eventhorizon/commit/556fa519072cb921cc1b2b55a7a5c454abedb085))
* remove golangci-lint backup config ([3fa9ad9](https://github.com/looplab/eventhorizon/commit/3fa9ad9c899bbd8e2e11e880e2f8bfbd035ff48a))
