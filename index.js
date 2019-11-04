const compose = require('docker-compose');
const path = require('path');
const Promise = require('bluebird');
const _ = require('lodash');
const Spinner = require('clui').Spinner;
const Docker = require('dockerode');

const docker = new Docker();

const pathToDockerComposeFile = path.join(__dirname, '..');
const dockerComposeOptions = {
  cwd: pathToDockerComposeFile,
  log: false,
};

const getServiceNames = async () => {
  return compose.configServices(dockerComposeOptions).then(result => {
    return result.out.split(/\n/).map(service => {
      return _.trim(service);
    }).filter(service => service);
  });
};

const getCurrentContainerId = async containerName => {
  return compose.ps({
    ...dockerComposeOptions,
    commandOptions: ['-q', containerName],
  }).then(result => {
    return _.trim(result.out);
  });
};

const getLatestImageId = async (containerName, containerObj) => {
  await compose.pullOne(containerName, {
    ...dockerComposeOptions,
    commandOptions: ['-q'],
  }).then(result => {
    return _.trim(result.out);
  });

  return containerObj.inspect().then(res => {
    return _.get(res, 'Config.Image', '');
  }).then(imageName => {
    if (imageName) {
      return docker.listImages({
        filters: {
          reference: [imageName],
        }
      }).then(ret => {
        return ret.reduce((acc, imageInfo) => {
          if (imageInfo.Created > acc.Created) {
            return imageInfo;
          } else {
            return acc;
          }
        }, { Created: 0 });
      });
    }
  }).then(latestImageInfo => {
    return _.get(latestImageInfo, 'Id', 'sha256:').split(':')[1];
  });
};

const getCurrentImageId = async containerObj => {
  return containerObj.inspect().then(res => {
    return _.get(res, 'Image', 'sha256:').split(':')[1];
  });
};

const updateContainer = async containerName => {
  const spinner = new Spinner(`Updating ${containerName}`);
  spinner.start();

  const currentContainerId = await getCurrentContainerId(containerName);
  if (!currentContainerId) {
    spinner.stop();
    console.log(`${containerName} is not running`);
    return;
  }

  const currentContainer = docker.getContainer(currentContainerId);
  const currentImageId = await getCurrentImageId(currentContainer);
  const latestImageId = await getLatestImageId(containerName, currentContainer);

  if (latestImageId && currentImageId !== latestImageId) {
    spinner.message(`Updating and restarting ${containerName}`);
    spinner.stop(); // Stop this after it's updated....
    console.log(`Updated ${containerName}`);
  } else {
    spinner.stop();
    console.log(`${containerName} is up to date`);
  }
};

(async () => {
  const serviceNames = await getServiceNames();
  Promise.mapSeries(serviceNames, async service => {
    await updateContainer(service);
  });
})();
