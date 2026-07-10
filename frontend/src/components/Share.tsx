import { useEffect, useState } from "react";
import QRCode from "qrcode";
import { Wifi, Smartphone } from "lucide-react";
import { getShareInfo } from "../services/api";

export default function Share() {
  const [urls, setUrls] = useState<string[]>([]);
  const [qrCodes, setQrCodes] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getShareInfo()
      .then(async (info) => {
        setUrls(info.urls);
        const codes: Record<string, string> = {};
        for (const url of info.urls) {
          codes[url] = await QRCode.toDataURL(url, {
            width: 220,
            margin: 1,
            color: { dark: "#D2E8DE", light: "#111816" },
          });
        }
        setQrCodes(codes);
      })
      .catch(() => setError("Failed to read network information."));
  }, []);

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <h2 className="text-2xl font-medium text-night-50 mb-2 flex items-center gap-2">
        <Wifi className="w-6 h-6 text-mint" /> Share on Wi-Fi
      </h2>
      <p className="text-sm text-night-400 mb-6">
        Other devices on the same Wi-Fi network — phones, tablets, other
        laptops — can use this Kibo. No internet needed: the network can be a
        local hotspot or router with no connection at all.
      </p>

      {error && <p className="text-red-400">{error}</p>}

      {!error && urls.length === 0 && (
        <div className="bg-night-850 border border-night-800 rounded-xl p-6 text-night-400">
          No Wi-Fi or LAN connection found. Connect this computer to a Wi-Fi
          network (or start a hotspot) and reopen this page.
        </div>
      )}

      <div className="space-y-4">
        {urls.map((url) => (
          <div
            key={url}
            className="bg-night-850 border border-night-800 rounded-xl p-6 flex flex-col sm:flex-row items-center gap-6"
          >
            {qrCodes[url] && (
              <img
                src={qrCodes[url]}
                alt={`QR code for ${url}`}
                className="rounded-lg border border-night-700 shrink-0"
                width={160}
                height={160}
              />
            )}
            <div>
              <p className="flex items-center gap-2 text-sm text-night-400 mb-2">
                <Smartphone className="w-4 h-4" /> Scan with a phone camera, or
                type the address in a browser:
              </p>
              <p className="text-xl font-medium text-mint-soft break-all">{url}</p>
            </div>
          </div>
        ))}
      </div>

      <p className="text-xs text-night-500 mt-6">
        Note: while this computer is on, anyone on the same network can open
        Kibo and see or change its data. Share only on networks you trust.
      </p>
    </div>
  );
}
