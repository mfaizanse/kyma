const axios = require('axios');
const https = require('https');
const httpsAgent = new https.Agent({
  rejectUnauthorized: false, // curl -k
});
axios.defaults.httpsAgent = httpsAgent;
const {
  testNamespace,
  kymaVersion,
  isSKR,
  skrInstanceId,
  backendK8sSecretName,
  backendK8sSecretNamespace,
  timeoutTime,
  slowTime,
  shootName,
  cleanupTestingResources,
  eventingSinkName,
  getClusterHost,
  checkFunctionReachable,
  deployEventingSinkFunction,
  waitForEventingSinkFunction,
  deployV1Alpha1Subscriptions,
  deployV1Alpha2Subscriptions,
  createK8sNamespace, eventingUpgradeSinkName, isUpgradeJob, isJSAtLeastOnceDeliveryTestEnabled,
} = require('./utils');
const {
  eventMeshSecretFilePath,
} = require('./common/common');
const {
  setEventMeshSourceNamespace,
} = require('../test/fixtures/commerce-mock');
const {
  info,
  debug,
  createEventingBackendK8sSecret,
} = require('../utils');
const {expect} = require('chai');

describe('Eventing tests preparation', function() {
  this.timeout(timeoutTime);
  this.slow(slowTime);

  it('Print test initial configs', async function() {
    debug(`Test namespace: ${testNamespace}`);
    debug(`Kyma version: ${kymaVersion}`);
    debug(`Is SKR cluster: ${isSKR}`);
    debug(`SKR instance Id: ${skrInstanceId}`);
    debug(`SKR shoot name: ${shootName}`);
  });

  it('Prepare SKR Kubeconfig if needed', async function() {
    // Skip this step if it is not a SKR cluster
    if (!isSKR) {
      this.skip();
    }

    if (!skrInstanceId) {
      info(`Skipping fetching SKR kubeconfig because skrInstanceId is not set.`);
      this.skip();
    }

    // 'skr-test/helpers' initializes KEB clients on import, that is why it is imported only if needed
    const {getSKRConfig} = require('../skr-test/helpers');
    const {initK8sConfig} = require('../skr-test/helpers');

    debug(`Fetching SKR config for Instance Id: ${skrInstanceId}`);
    const shoot = await getSKRConfig(skrInstanceId);

    debug('Initiating SKR K8s config...');
    await initK8sConfig(shoot);
  });

  it('Prepare EventMesh secret', async function() {
    // If eventMeshSecretFilePath is specified then create a k8s secret for eventing-backend
    // else skip this step and use existing k8s secret as specified in backendK8sSecretName & backendK8sSecretNamespace
    if (!eventMeshSecretFilePath) {
      this.skip();
    }

    debug('Creating Event Mesh secret');
    const eventMeshInfo = await createEventingBackendK8sSecret(
        eventMeshSecretFilePath,
        backendK8sSecretName,
        backendK8sSecretNamespace,
    );
    setEventMeshSourceNamespace(eventMeshInfo['namespace']);
  });

  it('Create test namespace', async function() {
    await createK8sNamespace(testNamespace);
  });

  it('Prepare eventing-sink function', async function() {
    debug('Preparing EventingSinkFunction');
    await deployEventingSinkFunction(eventingSinkName);
    await waitForEventingSinkFunction(eventingSinkName);
  });

  it('Eventing-sink function should be reachable through API Rule', async function() {
    const host = await getClusterHost(eventingSinkName, testNamespace);
    expect(host).to.not.empty;
    debug('host fetched, now checking if eventing-sink function is reachable...');
    await checkFunctionReachable(eventingSinkName, testNamespace, host);
  });

  it('Prepare v1alpha1 subscriptions', async function() {
    await deployV1Alpha1Subscriptions();
  });

  it('Prepare v1alpha2 subscriptions', async function() {
    await deployV1Alpha2Subscriptions();
  });

  it('Deploy eventing-upgrade-sink function', async function() {
    if (!isUpgradeJob || !isJSAtLeastOnceDeliveryTestEnabled) {
      debug(`Skipping deploy eventing-upgrade-sink function`);
      debug(`isUpgradeJob: ${isUpgradeJob}, 
          isJSAtLeastOnceDeliveryTestEnabled: ${isJSAtLeastOnceDeliveryTestEnabled}`);
      this.skip();
    }

    await deployEventingSinkFunction(eventingUpgradeSinkName);
    await waitForEventingSinkFunction(eventingUpgradeSinkName);
    debug(`checking if eventing upgrade sink is reachable through the api rule`);
    await checkFunctionReachable(eventingUpgradeSinkName, testNamespace, clusterHost);
  });

  afterEach(async function() {
    // if the test preparation failed, perform the cleanup
    if (this.currentTest.state === 'failed') {
      await cleanupTestingResources();
    }
  });
});
