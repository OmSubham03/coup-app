import type { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
  appId: 'com.coupgames.app',
  appName: 'Coup Games',
  webDir: 'www',
  server: {
    url: 'https://coup-server.greenstone-0a8cff3d.eastus.azurecontainerapps.io',
    cleartext: false
  },
  android: {
    allowMixedContent: false
  }
};

export default config;
