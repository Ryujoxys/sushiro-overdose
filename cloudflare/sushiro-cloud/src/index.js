// Deploy this tombstone before revoking the old OAuth and database secrets.
export default {
  async fetch() {
    return new Response(JSON.stringify({
      ok: false,
      error: "Cloud data access has been retired. Use local desktop data.",
      code: "cloud_data_retired",
    }), {
      status: 410,
      headers: {
        "content-type": "application/json; charset=utf-8",
        "cache-control": "no-store",
      },
    });
  },
};
