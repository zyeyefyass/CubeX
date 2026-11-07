import urllib.request
import re

try:
    with urllib.request.urlopen('http://10.0.0.186:5001/', timeout=2) as r:
        html = r.read().decode('utf-8', 'ignore')
        print("HTML length:", len(html))
        # Look for upload url / dropzone / action
        matches = re.findall(r'url\s*:\s*["\']([^"\']+)["\']', html)
        print("URLs in script:", matches)
        actions = re.findall(r'action\s*=\s*["\']([^"\']+)["\']', html)
        print("Actions:", actions)
except Exception as e:
    print("Error:", e)
