import { RenOSClient } from '@renos/sdk';

async function main() {
  const email = process.env.RENOS_EMAIL;
  const password = process.env.RENOS_PASSWORD;
  const server = process.env.RENOS_URL || 'http://localhost:8443';

  if (!email || !password) {
    console.error('RENOS_EMAIL and RENOS_PASSWORD must be set');
    process.exit(1);
  }

  const client = await RenOSClient.login(server, email, password);
  console.log('connected to RenOS');

  // List devices
  const devices = await client.listDevices(1, 10);
  console.log(`found ${devices.total} devices:`);
  for (const device of devices.data) {
    console.log(`  - ${device.name} (${device.hostname}) [${device.status}]`);
  }

  // Execute a command on the first device
  if (devices.data.length > 0) {
    const device = devices.data[0];
    console.log(`\nrunning command on ${device.hostname}...`);
    const cmd = await client.executeCommand(device.id, 'uname -a');
    console.log(`command ${cmd.id}: ${cmd.status}`);

    // Wait and check result
    await new Promise((resolve) => setTimeout(resolve, 2000));

    const result = await client.getCommand(device.id, cmd.id);
    console.log(`output: ${result.output}`);
  }

  // List workflows
  const workflows = await client.listWorkflows(1, 20);
  console.log(`\nworkflows: ${workflows.total}`);
  for (const wf of workflows.data) {
    console.log(`  - ${wf.name} (enabled: ${wf.enabled})`);
  }

  // List recent jobs
  const jobs = await client.listJobs(1, 10);
  console.log(`\njobs: ${jobs.total}`);
  for (const job of jobs.data) {
    console.log(`  - ${job.type} [${job.status}] ${job.completed}/${job.total}`);
  }

  // List active alerts
  const alerts = await client.listAlerts(1, 10);
  console.log(`\nalerts: ${alerts.total}`);
  for (const alert of alerts.data) {
    console.log(`  - ${alert.name} (${alert.severity}) [${alert.status}]`);
  }

  // List users
  const users = await client.listUsers(1, 20);
  console.log(`\nusers: ${users.total}`);
  for (const user of users.data) {
    console.log(`  - ${user.email} (${user.displayName}) [${user.role}]`);
  }

  // List groups
  const groups = await client.listGroups(1, 20);
  console.log(`\ngroups: ${groups.total}`);
  for (const group of groups.data) {
    console.log(`  - ${group.name} (${group.deviceCount} devices)`);
  }

  // WebSocket example
  console.log('\nconnecting to WebSocket...');
  const ws = client.websocket('/api/v1/ws');
  ws.on('message', (data: string) => {
    console.log('event:', data);
  });

  // SSE stream example
  console.log('\nconnecting to SSE stream...');
  const stream = client.streamEvents('/api/v1/deploy/stream');
  for await (const event of stream) {
    console.log('sse event:', event);
  }

  console.log('\ndone');
}

main().catch((err) => {
  console.error('fatal:', err);
  process.exit(1);
});
