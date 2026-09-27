import type { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
  appId: 'com.coupgames.app',
  appName: 'TimeCrusher',
  webDir: 'www',
  server: {
    url: 'https://coup-server.greenstone-0a8cff3d.eastus.azurecontainerapps.io',
    cleartext: false
  },
  android: {
    allowMixedContent: false,
    backgroundColor: '#020617'
  }
};

export default config;
