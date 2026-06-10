export async function listUsers() {
  return fetch("/api/users");
}
