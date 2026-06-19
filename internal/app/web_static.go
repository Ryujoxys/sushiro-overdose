package app

const logoBase64 = "iVBORw0KGgoAAAANSUhEUgAAAJ8AAACfCAYAAADnGwvgAAAACXBIWXMAACE4AAAhOAFFljFgAAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAAOdEVYdFNvZnR3YXJlAEZpZ21hnrGWYwAAEp9JREFUeAHtnV1y28i5ht8GRY0rnlOHVbEmjuVT02cFo1mBoctzZalOeWxdiV6B7BWQWoHlFZC+0thOlTR3uaO8AisrIKcycpxIqVEq8WRMiuj0BwISRfGnAXTjh+ynqm1JBAGw8eL7626QwYKDu5x3ve4ac1hFeIIzwb4WTFQgwMFQ8Tein8fAwM7ltufy9fOrn1kHTPxDvnzsOM750q2l481O5xyWazAsGG9WV9e8z3Adhm8ExJoUDJf/V2AYEqb871jIJoX5jpVZ57uTk2MsMHMvvrd377vwvAdCMFf+upaG0FQJLOWRtLQ/YBnHiybGuRPfAeeV3q+9DfTxQP66kSexKdCRgjxiJfbq0cefjjDnzIX4hgS3jZxZtwTMvRALLT5yqV5fPJQfojongpsECXG3XyofbX3sdDAnFFJ8b1ZWq/K/bSk4FwuGFGFzXqxhYcRHrvXil+6OTByezbmVU4Jcsvzv1XenJ00UlNyLz4puJr5LLqIIcy2+t1/dq1nRKdNxSs7TIrnjXIpvkEh4DfkjhyUSFBPKxGS3CIlJrsS3L4e5Sv1eYxETCd3IEZz6o7992EWOyY34rIs1Qq7jwczFJ8smVBQmF7sGixHy6oozFd+blfs7At4eLGnQ8YTzdOssPwlJJuKzsV125CkWTF18NpPNHipQSzf8NGs3nKr4rJvNFR2vtLyepQAdpMTrldUXVni5gjv9bvv73/3+GTLCuOXzpzt96h0UPb67nCI/iih++JBVHGhUfJRYyLurhSLEdwwdeDgWjP1IU90dzzn3+qXjWxWcq6y/2K/IJKrcq/QFqzCINbk/Lt3KNyjM/EK29/j05DlSxJj48iy8cD2FJ/CDADu+9aXZBT77d+9yp1deE47nyg5/ABJnDqF6oCxIP0VKGBFfHoVHgvOAV3IU5dC02GZBYkS/5DpCPJQntoEcQZlw+XZ5M43+0S6+PAlvWHB5Kq4OQzHxr596G6V8TY49Xr69vG5agFrFlxfh0d1LLvWLL8vNIq2X9d1z36nLD/Ag60SG+lC64HUYRJv48iA8v3gq2G5erVwU9ldWqw4TtSxFaDoG1CK+rIU3T6IbJXsRmsuCtYjv9cq998hiVoosj3hevgbLTbF/517dcbCdhQhN1QETi49GLmTpINUquZ9ICG/3ydlfFmrE5DImhL8+OWVYVVrAV9BIIvHRBFAZ2NeRIoNB8b4cFP/YwYKyf+e+6zheI2UrSA9CWpcxoLZHesQWXzA7pYWUWFRrN4mDCq98LvXqzBE7SI+OLMF8q6uCEEt8GSQYx17J21xkazeJtBMSnSWYWLNapPAOkJLw5PDXSyp4WuGNZ+v0pOk53ro/Np0CVAjXNRMmsuVLM84TQjy3blaNtN2wtIDfJo3/IokvcLdtGIbiO1m321yEEopu/JIMQw3mSRz/RXK7QZxnloH7WLfCi8fW2Ye69ExpzM3j3U+9RCJXFh+5W5iO86hoLOMXnen8IuILECyFqVHiGZV9EBMl8ZG7NR7nBcKziYUe/EQkBQE6zF8MFu+9ShuZdrdWeEZISYD89Vf36ojBTPEFD2LkMIRfPLbCM4YvQNMxoMAOzUtERKaKj3Yo6zpGMyc/q7XCM0oKSQgtEnuBiEwVHz2UEQatHtXxbFabDiRAaaEOYQhppKpRk4+J4guSjCoMQSMXtoCcLssXy09NjoSUWDQvOVF8pX7PXGlFdsCT05PMFisvKpvnHT++DlbvaYeG3qJYv7HiI6tHZhQmCDJbWDKB4muT8V8U6zdWfIHVM4JH091tgpEpT85O9kzFf1Gs3w3xBVbPhQHkQHJzq8CP7p8nKP4z5X5Vrd8N8Umr58JErCfdbb/kpTHmaFGA4j/hOUYK0L71k0Zs1nbOmDcacbnW3eaPx3//82HwZTLacbxudeY2w7+8/u3/0KMbOHQjrZ51t/mE1sMYcb8Kox7XxMeYMLIqyjNk3i3JIW8kmHgJ/VQ+//tzddoGl+LzEw3maX9ojZ9k2FGMXLPcXd4zYf0cz3k49fXwhyDR0I5NMvKPn3wYsH6DxOMun/S6c7UhtM/9962eTTIKgUHrV534Gv0zSIv1P7DQWr3iYMr6McEeTHptyf/34oIzxprQe9DO448fOrAUBrJ+vXLva2iGst4iParOYrFYLBaLxWLJE/b7gQtCVTZaaMKhB655f1Gh2qjxx4dYkkOzZcRQ0zGN3g32RQIgEVKtkgdtIzhGPfhZNxvBseuw5BKOgbWjR6qJMU3H9K29CfsebfTcaF0ukkT+M6zV004V0S8Sx5Wloccn0IWmi6MiihaSuc1KhGPVkRw+dLzqlG02gtddmHtYuotBfx8EPxca6qTQjbkjr9FF5hh0aByRTWttJBPghuJxfkYy68cxONfwnEepYXJ/0PYN6IlPOQY37egxdO0/M4Y/FHUkCayN5AJTEUaS2KyleBwX8eC43g/VkdefKR4/yo0W3vBkFEJLSrHttBv+Z+iJpzPBhXmhmRKHihWOs1CdRNAe2kdrwnZ1qH3GBgZCqgbveRH8jfar62ZvoaBWkE5cZNTaiO8aVaxPC9Ggc3k/sg8+Zfss+25cX7ooGBzJY7kwG4zz3iQBenPGvqO6pNFMvTFjexfZCG1a20bBUI1hxomujmQXgSMZVdwUfhvRs90Gop9blOw7rZY00cqEUXejIrowSG4jWgeF+6hCH2RBXcQTcw03z6+h+N4W8iG64Wai0G4Ujtl38bDowve0EV10w/sYJYy7qkiHGpDIIqsWvif1BfUffV5y+Y1gf3Vclbmi7rONgiYfkz7sJMFEsZbUWlDrmHaw/UFwTi6uyg91DDLGKpJTw/jzrKvvQqnuWMX1IjRX27W/XZT+fY+CCi+kioFI2sH/dPHHWSkSQJSOiVL2UC0kv0d8tqHHclQw+zzjwgHl/o163oWlhmiupYrotBT3X0d0pok7TtG2DTPic6Hez3WkD8fV6FcDKYhf1SqFd2PcUgqHWibZQjTWMH1YLA6TJlOkKT4+5v1V6I2dOQb9V8X44cU2JghwCXoO3lDc9li2Tdk6iEdHNlreV5uxnYvBeXUwG9qOxDop2Ym7/PNHmEG1ZNLE+M+/jUH/UB9SXx6P2f9w8vjfI79XRtos6D3Uv+uA3kfycqhntnvQV2tSSWqqCvvhM84/rtVDcHwTlk812+UT3u8qvl93a0NjrXF0vHNa0z3QzRWO7WrYRxXxWZux77iolHEaM/ZRV9iHiRb56xImMSumoaa7aDxKHeMFVJ/xPpUb5wDJmJXxxqWF2f3OFfZTV9iPieYiIbUIB2vDfNbj4iqQnmXax00UiHsBZ9GGfvFN2ye1RoR9NZG++FpIwE6CA9eQfd1JxWI3oIdpx4oDh/6bpgUoXz8dLbZHmRXHUKP4ronplrCGbGhArYM49DAtPovDrJJWC9GpTDjPdrC/RvA6XddqcA4urhZqcYX90zZrQYuVdHDMzgyH63ezLEwb6Y3VEjWoCa8OfUzLTOPQwPRzdxGfCq5EpS0r1cGsAL2Fm3eAajY8KlpVSEzVCNuqCK8NvUyzVHGYVmBvYU6hDzbpQ08bn3WhdtGpNRDN3YUXoj3jvbUI51CFXjiSiy90W7Ni7SrmkEkXT7WMEmV6EQnJhRocNy1rAwNrw3H1NIMoxzbBtJuNQpMWrq/doBZnMmrcIcvcUsPkC8UV9xFnZu+24r459K2s4zCDrvOb1X42+BlSZ1K80kT0gDTO5EpXcd8c8S7WqBUyhUppR1drYw4eXMQx3lrFHSZTKdGMthbUaSHZBeMwR5JZzXFaHQWGY/zim6QxRQvRO1L1mHGmloetCrMkObe4N1NhGR16akGPKY8y5y+qMFzk90LF+dxJ2h4Kwuh8vvAxYyE0l60OPRxiMJ+LR3hPR3G7Y9nOEf0mWYd5OhP+diTbPzA475DziL+HZRhq38j2DoOYvHDUcHX3ULznQj9VmLNKdUSzEHWkRwsFtU5pMFzApI7iMMfohdAljijzC6MKOykcA4tEn70OyyUcV5ltGnekikiaiMeawr7pdQ5LLqCLYcrNToIE2MRNYYRrgZPAMXlWjWmrbokAwyAbO8L1QDYtOK6m2ZxrPo8Kri/GPoLmBSwWi8VisVgsFotlOpRw4O2d+65gYhsaoe/bfXT20y4shYG+9LvU79Wgme9OT56O+7s/vHaxdNFx+pO/jjwOUszYv3P/3dbZT0ewFIIlr7vt6Z5owW48juMS/2vutz5+7DCwI2imxIT2u8hiBrJ6ntA/w0caoVeTXnPCH+SBf4DuA0O40vq5sOQesnowUICX4juc9Nql+L64KDdhAGv98o8pq0felLzqpNcvxbd53jk34Xqt9cs/pqxeH3g17XXn2saCGclOrfXLL4HVq0M3DJ1bt8uH0za5Jj4/M2X6xz/J+n3/u9/HXf9hMYiJ0gohix1Hm53O1HF6Z8zfpprK2CfjsdoB54VfWTVPvFlZrUrDUIUB+iVvphe9Ib7l7vKejP1MzHCpdH/pNmDJBeRupfDMWD2gOS3RCLkhPko8ZHr8EiYQ2Hi9sqp1JMUSj8DdchigLxwl7znO7Zq0fhKxR3cdLJnxZuX+jil365dXFEe1xorPqPWT7tfpd5M+ctYSk8GNL+owRJSKiTPpBbPWD2vS/Wp7OLRFDUr45I3fklbPTOLHcBhlLH+i+AxbP4l4Zssv6dL9l5/wcRjCc7znUbZ3pr1I1s9E3S9Ell9e2NGPdHj71b2avJYbMIRqhjvMVPGR9fMMjXpcngDzDmwCYhYSnpFRjAAKz1TqeqM4szbYOj1pmhjzHcKPQ6wAzUCZrUnhEX0hXka1esRM8QVE8uUx4FaA+vmDrKkKeGYfBCDDsq2zD3XEQEl8352eHMuDmJ4SbwWoERJeH6IJw8gkI/bDllQtn/HkI8AKUAMU46UhPAEWy92GKIvPn+8n2CbMQwJ8//qre8Yys3nGdHJxiTREX9wu15EAFnF7fH9n9RljIpUCscNQf/S3D3YFnAJUQO790n0hRDpfieCVvP9NYvWIyOIj3qysUpXcRSqwvcenJ6YTnkJDYUowZJnK1yFIy7obN8kYJpb49u/e5Y7ntKTT50iHjldaXt/62OnAcg0KT2Q41DA2ZDaKwOHjsw9awi/lmG8YMree5zxFevhxoB2Ou4LcrD8+LnCQmvBknOctedq8UCzLF5Jm/BciC97Nfqm8u8hW8O3d+67X94yO044yGMXof5s0zru+z4R8/9vVPeaIHaTLuXDE7pO//mWhnm3sJxWfei9MzcWbhhDi+ZMzvf2dWHyENP/v5emlEuyOQE9a2JVF8CbmGBLdxS/dHSHYs9Rc7BC6EoxRtIjvoMIr3eXu+xQTkGvQ2DMrsd1HH+fvuTDBIh9jU95nYUp4hBbxERlkwDeYFxGSpfv8r15VxtMUznBkhAe82jr9UIUhtImPyIMAAwrpjrN2r8PQjSz7L/a4rdoxNJMjASJYBnAoreGrvFpDP4n4tbeBPrbTK9zP5Hj59vL6rEXfSdEuPiJPAhyiE7jlzIU4LDj561rWVm4Y6qPy7fKmaeENjmWInArwEupkwcQ7x3GOlm4tHZvs7AM5/NX1umsQ7AEG1i2LysBMTMd4oxgTH5F3AY5wHEwZ+xNzGFnJjiiJ83K53FERJgmsV+pVRE9w4THOGL4GE1wOfbl5smyToOlRT05PUh1BMio+wi/DlHutjOqA2qD4UVrKmyIsxo01FZPllGkYF19IRiMhlin4Q2Yenm/9/aSJDCghJf7w73/+8f9/819MuiMXluxhfgL2f0/OTv6IjEjN8oXIiv2adF8H8+CuigolW/1S/6nOSQLxziMD/ETkwnlhchGzZTxZxXfjyER8Ift37tUdBvvI3DSguXie8zRP34uSqfiIgpVjCgmVUWixTxqF4yhkLr4QawUNkENrN0ysafQmoDiEVkTB0DOhFw2K7ZZ/s/xtnr9+LDeWb5j9ldWqQ1+fYF1xZPKSyaqQS/GFWBGq44tOsN0ifdFirsUX4seDDratCG9SRNGFFEJ8BE1D+vVTb8NawgFFFl1IYcQ3jO+OgZ2iT1aIwzyILqSQ4gvxh+ognskL8rAI05bi4k8AEOLlrS+X9/JWq0tCocU3jG8NhXg4L0N2JDia3CkEO5zXb2ufG/GFXMaGQjxkrBgTOS+RRWEpth/mWXDDzJ34RqGn3TuO5zLBHuRogY6PP0FViCM5/PVOXJQPt84X6xEgcy++UUiMTCYqjMkG9jVSWsATrKQ7lq70T4Km7PfKR4smtlEWTnzj2K9wjqULTmsunMG6CxIlp9eE/N3faEJ5Z3h6vXyfdJviXLp7+tuPnvy9JIVWvq22DmTR+A9Rn65p9nGmSQAAAABJRU5ErkJggg=="

const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="sushiro-csrf" content="{{CSRF_TOKEN}}">
<link rel="icon" type="image/png" href="data:image/png;base64,` + logoBase64 + `">
<link rel="apple-touch-icon" href="data:image/png;base64,` + logoBase64 + `">
<title>SUSHIRO Overdose</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
:root{
  --red:#B81C22;--red-dark:#9F1419;--red-soft:#FFF1F1;
  --ink:#191817;--text:#282522;--sub:#66615C;--mute:#76716B;
  --paper:#FFFFFF;--wash:#F5F3F1;--line:#E5E0DB;--line-strong:#D5CEC7;
  --green:#21823F;--green-soft:#ECF7EF;--yellow:#B67800;--yellow-soft:#FFF5D8;
  --blue:#2B5B83;--blue-soft:#EEF5FA;--shadow:0 12px 34px rgba(42,35,28,.08);
  --font:"PingFang SC","Hiragino Sans GB","Microsoft YaHei",-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
}
body{min-height:100vh;background:radial-gradient(circle at 1px 1px,rgba(184,28,34,.035) 1px,transparent 1.7px) 0 0/26px 26px,linear-gradient(180deg,#fff 0,#f7f5f2 260px,var(--wash) 100%);color:var(--text);font-family:var(--font);-webkit-font-smoothing:antialiased}
button,input,select{font:inherit}
.topline{height:4px;background:var(--red)}
.shell{max-width:1120px;margin:0 auto;padding:0 24px}
.hdr{position:sticky;top:0;z-index:20;background:rgba(255,255,255,.92);backdrop-filter:blur(18px);border-bottom:1px solid var(--line)}
.hdr-in{height:72px;display:flex;align-items:center;gap:18px}
.brand{display:flex;align-items:center;gap:12px;min-width:0}
.brand img{width:42px;height:42px;border-radius:50%}
.brand strong{display:block;font-size:16px;line-height:1;color:var(--ink)}
.brand span{display:block;margin-top:4px;font-size:11px;color:var(--mute);letter-spacing:.08em;text-transform:uppercase}
.nav{margin-left:auto;display:flex;gap:4px;padding:5px;background:#F0EDEA;border:1px solid var(--line);border-radius:999px}
.nav a{display:inline-flex;align-items:center;height:34px;padding:0 14px;border-radius:999px;color:var(--sub);font-size:13px;font-weight:700;text-decoration:none;white-space:nowrap}
.nav a.on{background:var(--paper);color:var(--red);box-shadow:0 2px 10px rgba(32,25,18,.08)}
.subnav{margin:0 0 18px;background:transparent;border:0;padding:0;gap:6px;flex-wrap:wrap;justify-content:flex-start}
.subnav a{background:#F0EDEA;border:1px solid var(--line)}
.subnav a.on{border-color:var(--red)}
.ver{margin-left:6px;padding:7px 11px;border-radius:999px;background:var(--ink);color:#fff;font-size:11px;font-weight:700}
.mode-switch{display:flex;gap:3px;padding:4px;background:#F0EDEA;border:1px solid var(--line);border-radius:999px}
.mode-switch button{height:28px;padding:0 10px;border:0;border-radius:999px;background:transparent;color:var(--sub);font-size:12px;font-weight:900;cursor:pointer;white-space:nowrap}
.mode-switch button.on{background:#fff;color:var(--red);box-shadow:0 2px 8px rgba(32,25,18,.08)}
.mode-settings{display:flex;align-items:center;justify-content:space-between;gap:14px;padding:14px 16px;border:1px solid var(--line);border-radius:12px;background:#FBFAF8}
body.simple-mode .advanced-only,body.simple-mode .advanced-page{display:none!important}
body.advanced-mode .simple-only{display:none!important}
.wrap{padding:30px 0 80px}
.grid{display:grid;grid-template-columns:minmax(0,1fr) 320px;gap:18px;align-items:start}
.hero{min-height:250px;background:var(--paper);border:1px solid var(--line);border-radius:10px;padding:30px;box-shadow:var(--shadow);position:relative;overflow:hidden}
.hero:before{content:"";position:absolute;inset:0 0 auto 0;height:6px;background:var(--red)}
.hero:after{content:"";position:absolute;right:-80px;bottom:-120px;width:280px;height:280px;border-radius:50%;background:radial-gradient(circle,rgba(184,28,34,.07) 0,rgba(245,186,36,.05) 55%,transparent 72%);pointer-events:none}
.eyebrow{display:inline-flex;align-items:center;gap:8px;padding:6px 10px;border-radius:999px;background:var(--red-soft);color:var(--red);font-size:12px;font-weight:800}
.hero h1{margin-top:22px;font-size:34px;line-height:1.15;letter-spacing:0;color:var(--ink);max-width:560px}
.hero p{margin-top:12px;color:var(--sub);font-size:15px;line-height:1.8;max-width:620px}
.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:26px}
.bt{display:inline-flex;align-items:center;justify-content:center;height:42px;padding:0 20px;border:0;border-radius:999px;cursor:pointer;font-size:14px;font-weight:800;text-decoration:none;transition:transform .14s,box-shadow .14s,background .14s,border-color .14s}
.bt:hover{transform:translateY(-1px)}
.bt:disabled{opacity:.45;cursor:not-allowed;transform:none}
.bt-l{height:48px;padding:0 28px;font-size:15px}
.bt-s{height:34px;padding:0 14px;font-size:12px}
.bt-r{background:var(--red);color:#fff;box-shadow:0 10px 22px rgba(184,28,34,.22)}
.bt-r:hover{background:var(--red-dark)}
.bt-y{background:#F5BA24;color:#2C2418;box-shadow:0 8px 18px rgba(190,128,0,.16)}
.bt-o{background:var(--paper);color:var(--red);border:1px solid rgba(184,28,34,.35)}
.bt-w{background:var(--paper);color:var(--text);border:1px solid var(--line-strong)}
.side{display:flex;flex-direction:column;gap:14px}
.card,.cd{background:var(--paper);border:1px solid var(--line);border-radius:10px;padding:20px;box-shadow:0 8px 24px rgba(42,35,28,.05);margin-bottom:18px}
.card h2,.cd-t{font-size:12px;letter-spacing:.09em;text-transform:uppercase;color:var(--mute);font-weight:900;margin-bottom:14px}
.engine{border-radius:10px;border:1px solid var(--line);padding:16px;background:#FBFAF8}
.engine .row{display:flex;align-items:center;gap:10px}
.dot{width:9px;height:9px;border-radius:50%;background:var(--mute);box-shadow:0 0 0 4px rgba(155,150,145,.12)}
.engine strong{font-size:15px;color:var(--ink)}
.engine p{margin-top:8px;color:var(--sub);font-size:12px;line-height:1.6}
.engine.capturing .dot{background:var(--yellow)}
.engine.booking .dot,.engine.sniping .dot{background:var(--blue)}
.engine.success .dot{background:var(--green)}
.engine.error .dot{background:var(--red)}
.notice{margin-top:16px;padding:13px 14px;border-radius:10px;background:var(--yellow-soft);border:1px solid #ECD681;color:#6F4B00;font-size:13px;line-height:1.6}
.ps{font-size:13px;line-height:1.9;color:var(--sub)}
.ps b{color:var(--ink)}
.ps .line{display:block;margin-top:4px}
.cg{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:8px}
.ci{display:flex;align-items:center;gap:8px;min-height:38px;padding:9px 10px;border-radius:8px;background:#F3F0ED;color:var(--sub);font-size:12px;font-weight:700}
.ci:before{content:"";width:8px;height:8px;border-radius:50%;background:#C9C1BA}
.ci.ok{background:var(--green-soft);color:var(--green)}
.ci.ok:before{background:var(--green)}
.ci.bad{background:var(--red-soft);color:var(--red)}
.ci.bad:before{background:var(--red)}
.ci.warn{background:var(--yellow-soft);color:var(--yellow)}
.ci.warn:before{background:var(--yellow)}
.fg{margin-bottom:16px}
.fg label{display:block;margin-bottom:6px;color:var(--sub);font-size:12px;font-weight:800}
.fr{display:flex;gap:12px;flex-wrap:wrap}
input[type=number],input[type=text],input[type=time],input[type=date],select,textarea{width:100%;height:40px;padding:0 12px;background:#fff;border:1px solid var(--line-strong);border-radius:8px;color:var(--ink);font-size:14px}
input[type=number]{width:86px}
textarea{height:88px;padding:10px 12px;resize:vertical;line-height:1.5}
input:focus,select:focus,textarea:focus{outline:0;border-color:var(--red);box-shadow:0 0 0 3px rgba(184,28,34,.08)}
input::placeholder,textarea::placeholder{color:var(--mute);opacity:.85;font-weight:400}
.settings-grid{display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1fr);gap:18px}
.settings-wide{grid-column:1/-1}
.status-card{position:relative;overflow:hidden;min-height:136px;padding:16px;border:1px solid var(--line);border-radius:14px;background:linear-gradient(145deg,#fff 0,#FBFAF8 78%);box-shadow:0 10px 24px rgba(42,35,28,.05)}
.status-card:before{content:"";position:absolute;inset:0 auto 0 0;width:5px;background:#C9C1BA}
.status-card.ok:before{background:var(--green)}
.status-card.warn:before{background:var(--yellow)}
.status-card.bad:before{background:var(--red)}
.status-card b{display:block;color:var(--ink);font-size:15px;margin-bottom:6px}
.status-card strong{display:block;color:var(--ink);font-size:22px;line-height:1.1;font-weight:950;letter-spacing:-.02em}
.status-card p{margin-top:8px;color:var(--sub);font-size:12px;line-height:1.6}
.status-card .fl{margin-top:12px}
.status-priority{border-color:#E7C1C3;background:linear-gradient(135deg,#FFF6F6 0,#fff 55%,#FBFAF8 100%)}
.debug-strip{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:14px 16px;border:1px dashed var(--line-strong);border-radius:14px;background:#FBFAF8}
.debug-strip b{display:block;color:var(--ink);font-size:14px}
.debug-strip span{display:block;margin-top:4px;color:var(--sub);font-size:12px;line-height:1.55}
.debug-only{display:none!important}
.tl{display:flex;flex-direction:column;gap:8px}
.tr{display:flex;align-items:center;gap:8px}
.tr input{width:82px;text-align:center}
.tr .sp{color:var(--mute)}
.tr .x{display:inline-flex;align-items:center;justify-content:center;width:28px;height:28px;border-radius:999px;color:var(--red);cursor:pointer;background:var(--red-soft);font-weight:900}
.at{display:inline-flex;margin-top:8px;color:var(--green);font-size:13px;font-weight:800;cursor:pointer}
.chips{display:flex;gap:8px;flex-wrap:wrap}
.chip{display:inline-flex;align-items:center;min-height:34px;padding:0 13px;border:1px solid var(--line-strong);border-radius:999px;background:#fff;color:var(--sub);font-size:12px;font-weight:800;cursor:pointer}
.chip.on{background:var(--red);border-color:var(--red);color:#fff}
.check{display:inline-flex;align-items:center;gap:8px;height:40px;padding:0 12px;border:1px solid var(--line-strong);border-radius:999px;background:#fff;color:var(--sub);font-size:13px;font-weight:800;cursor:pointer}
.check input{width:auto;height:auto}
.store-list{display:grid;gap:8px}
.store-row{display:grid;grid-template-columns:auto minmax(0,1fr) auto auto;align-items:center;gap:8px;padding:10px 12px;border:1px solid var(--line);border-radius:10px;background:#FBFAF8}
.store-row b{font-size:13px;color:var(--ink);overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.store-row span{font-size:11px;color:var(--mute)}
.ico{display:inline-flex;align-items:center;justify-content:center;width:28px;height:28px;border:1px solid var(--line-strong);border-radius:999px;background:#fff;color:var(--sub);cursor:pointer;font-size:13px;font-weight:900}
.preset-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(132px,1fr));gap:8px;margin-bottom:16px}
.preset{min-height:42px;padding:0 12px;border:1px solid var(--line-strong);border-radius:10px;background:#FBFAF8;color:var(--text);font-size:12px;font-weight:900;cursor:pointer}
.sample-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:12px}
.sample-state{display:grid;grid-template-columns:repeat(auto-fit,minmax(170px,1fr));gap:10px;margin-top:14px}
.metric{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:10px;margin-bottom:14px}
.dash-head{display:grid;grid-template-columns:1fr auto;gap:16px;align-items:end;margin-bottom:16px}
.dash-title{font-size:28px;line-height:1.1;color:var(--ink);font-weight:950;letter-spacing:-.02em}
.dash-copy{margin-top:8px;color:var(--sub);font-size:13px;line-height:1.7;max-width:720px}
.dash-controls{display:flex;gap:8px;flex-wrap:wrap;justify-content:flex-end}
.dash-controls select{width:auto;min-width:112px}
.dash-target{display:flex;align-items:center;gap:7px;height:34px;padding:0 10px;border:1px solid var(--line-strong);border-radius:999px;background:#fff;color:var(--sub);font-size:12px;font-weight:900}
.dash-target input{width:104px;height:28px;padding:0 8px;border:0;background:#F7F4F1;border-radius:999px;font-weight:900}
.advisor-panel{margin-bottom:16px}
.advisor-card{position:relative;overflow:hidden;display:grid;grid-template-columns:minmax(0,1.25fr) minmax(240px,.75fr);gap:14px;padding:18px;border:1px solid var(--line);border-radius:16px;background:linear-gradient(135deg,#191817 0,#332A23 58%,#6F251F 100%);color:#fff;box-shadow:0 18px 36px rgba(42,35,28,.13)}
.advisor-card:after{content:"";position:absolute;right:-70px;top:-90px;width:220px;height:220px;border-radius:999px;background:rgba(255,255,255,.08)}
.advisor-card.warn{background:linear-gradient(135deg,#5A3210 0,#9B6114 62%,#C8881A 100%)}
.advisor-card.bad{background:linear-gradient(135deg,#451316 0,#8F171D 64%,#B81C22 100%)}
.advisor-card.muted{background:linear-gradient(135deg,#4A443E 0,#6B625B 100%)}
.advisor-main{position:relative;z-index:1}
.advisor-eyebrow{display:inline-flex;align-items:center;height:24px;padding:0 9px;border-radius:999px;background:rgba(255,255,255,.14);font-size:11px;font-weight:900;color:rgba(255,255,255,.82)}
.advisor-main h3{margin:10px 0 0;font-size:28px;line-height:1.12;letter-spacing:-.03em}
.advisor-main p{margin:9px 0 0;color:rgba(255,255,255,.78);font-size:13px;line-height:1.65}
.advisor-milestones{position:relative;z-index:1;display:grid;gap:8px}
.advisor-point{display:grid;grid-template-columns:auto 1fr auto;gap:10px;align-items:center;padding:10px 11px;border-radius:12px;background:rgba(255,255,255,.12);backdrop-filter:blur(8px)}
.advisor-point span{font-size:11px;font-weight:900;color:rgba(255,255,255,.72)}
.advisor-point b{font-size:15px;color:#fff}
.advisor-point strong{font-size:20px;color:#fff;font-variant-numeric:tabular-nums}
.kpi-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(120px,1fr));gap:10px;margin-bottom:16px}
.kpi{padding:16px 14px;border:1px solid var(--line);border-radius:12px;background:#fff;box-shadow:0 8px 20px rgba(42,35,28,.04)}
.kpi span{display:block;color:var(--mute);font-size:12px;font-weight:900;margin-bottom:7px}
.kpi strong{display:block;color:var(--ink);font-size:28px;line-height:1;font-weight:950;letter-spacing:-.03em}
.kpi p{margin-top:8px;color:var(--red);font-size:12px;font-weight:800}
.dash-chart{position:relative;min-height:300px;border:1px solid var(--line);border-radius:12px;background:linear-gradient(180deg,#fff 0,#FBFAF8 100%);padding:14px;overflow:auto}
.dash-chart svg{width:100%;min-width:820px;height:280px;display:block}
.data-source{display:grid;gap:6px;padding:12px 13px;border:1px solid var(--line);border-radius:10px;background:#F7F3EE;color:var(--sub);font-size:12px;line-height:1.65}
.data-source.ok{background:var(--green-soft);border-color:#BFE4CC}
.data-source.warn{background:var(--yellow-soft);border-color:#E9D08A}
.data-source.bad{background:var(--red-soft);border-color:#F0B7B9}
.data-source b{color:var(--ink);font-size:13px}
.data-source p{margin:0}
.data-source-lines{display:grid;gap:3px}
.curve-sampling{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:14px;align-items:center;margin:12px 0 16px;padding:16px;border:1px solid var(--line);border-radius:14px;background:linear-gradient(135deg,#FFF9F4 0,#FBFAF8 58%,#F5F3F1 100%)}
.curve-sampling b{display:block;color:var(--ink);font-size:16px}
.curve-sampling p{margin-top:5px;color:var(--sub);font-size:12px;line-height:1.65;max-width:760px}
.curve-sampling .sample-state{margin-top:10px}
.curve-sampling-actions{display:flex;align-items:center;gap:8px;flex-wrap:wrap;justify-content:flex-end}
.switch{display:inline-flex;align-items:center;gap:8px;height:34px;padding:0 12px;border:1px solid var(--line-strong);border-radius:999px;background:#fff;color:var(--sub);font-size:12px;font-weight:900;cursor:pointer}
.switch input{width:auto;height:auto;accent-color:var(--red)}
.dash-split{display:grid;grid-template-columns:1.15fr .85fr;gap:16px;margin-top:16px}
.dash-tip{position:fixed;z-index:20;max-width:260px;padding:9px 10px;border-radius:9px;background:#191817;color:#fff;font-size:12px;line-height:1.55;box-shadow:0 10px 24px rgba(25,24,23,.22);pointer-events:none;white-space:pre-line}
.chart-hot{cursor:crosshair}
.called-table{width:100%;border-collapse:separate;border-spacing:0;font-size:12px;min-width:720px}
.called-table th,.called-table td{padding:9px 10px;border-bottom:1px solid var(--line);text-align:left;white-space:nowrap}
.called-table th{color:var(--mute);font-size:11px;letter-spacing:.06em;text-transform:uppercase}
.called-table strong{color:var(--red);font-size:16px}
.called-table .num{font-variant-numeric:tabular-nums;font-weight:900;color:var(--ink)}
.heat-wrap{overflow:auto;border:1px solid var(--line);border-radius:12px;background:#fff}
.heat{width:100%;border-collapse:separate;border-spacing:0;font-size:12px;min-width:760px}
.heat th,.heat td{padding:8px;border-bottom:1px solid var(--line);text-align:center;white-space:nowrap}
.heat th:first-child,.heat td:first-child{position:sticky;left:0;background:#fff;text-align:left;font-weight:900;color:var(--sub);z-index:1}
.heat-cell{display:block;border-radius:8px;padding:7px 6px;font-weight:900;color:#2C2418;background:#F5F3F1}
.heat-cell.hot{background:#B81C22;color:#fff}
.heat-cell.warm{background:#F5BA24;color:#3A2A10}
.heat-cell.mild{background:#FFE8B0;color:#6F4B00}
.rank-list{display:grid;gap:8px}
.rank-row{display:grid;grid-template-columns:32px 1fr auto;gap:10px;align-items:center;padding:11px 12px;border:1px solid var(--line);border-radius:12px;background:#FBFAF8}
.rank-row b{display:block;color:var(--ink);font-size:13px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.rank-row span{display:block;color:var(--mute);font-size:11px;margin-top:2px}
.rank-row strong{font-size:18px;color:var(--red)}
.weekday-strip{display:grid;grid-template-columns:repeat(auto-fit,minmax(130px,1fr));gap:8px}
.weekday-card{border:1px solid var(--line);border-radius:12px;background:#FBFAF8;padding:12px}
.weekday-card b{display:block;color:var(--ink);font-size:14px}.weekday-card span{display:block;margin-top:5px;color:var(--sub);font-size:12px;line-height:1.55}
.chart{min-height:260px;padding:16px;border:1px solid var(--line);border-radius:10px;background:#FBFAF8;overflow:auto}
.chart svg{width:100%;min-width:680px;height:260px;display:block}
.chart-grid{stroke:#E5E0DB;stroke-width:1}
.chart-axis{stroke:#BDB5AD;stroke-width:1.2}
.chart-label{fill:var(--mute);font-size:11px;font-weight:700}
.chart-axis-title{fill:var(--mute);font-size:11px;font-weight:800}
.chart-legend{display:flex;gap:12px;flex-wrap:wrap;margin:10px 0 0;color:var(--sub);font-size:12px;font-weight:800}
.legend-line{display:inline-flex;align-items:center;gap:6px}
.legend-line:before{content:"";width:18px;height:3px;border-radius:999px;background:var(--red)}
.legend-line.global:before{background:var(--blue)}
.legend-history{display:inline-flex;align-items:center;gap:6px}
.legend-history:before{content:"";width:22px;height:0;border-top:3px dashed var(--blue)}
.legend-band,.legend-now,.legend-mine,.legend-pressure,.legend-turso-trend{display:inline-flex;align-items:center;gap:6px}
.legend-band:before{content:"";width:18px;height:10px;border-radius:3px;background:rgba(212,156,39,.32)}
.legend-now:before{content:"";width:18px;height:3px;border-radius:999px;background:var(--red)}
.legend-mine:before{content:"";width:18px;height:0;border-top:2px dashed var(--red)}
.legend-pressure:before{content:"";width:18px;height:10px;border-radius:3px;background:rgba(120,120,120,.28)}
.legend-turso-trend:before{content:"";width:22px;height:0;border-top:3px dotted var(--green)}
.answer-card{border:1px solid var(--line);border-radius:14px;background:linear-gradient(180deg,#fff 0,#FBFAF8 100%);padding:16px;margin-bottom:14px}
.answer-lead{font-size:17px;font-weight:800;line-height:1.5;color:#1f1b18}
.answer-chips{display:flex;gap:8px;flex-wrap:wrap;margin-top:12px}
.answer-chip{display:inline-flex;flex-direction:column;gap:2px;border:1px solid var(--line);border-radius:10px;padding:6px 10px;min-width:84px;background:#fff}
.answer-chip span{font-size:11px;color:var(--sub);font-weight:700}
.answer-chip strong{font-size:15px}
.press-low{color:#1f8a4c}.press-medium{color:#B67800}.press-high{color:#c4561a}.press-extreme{color:#b81c22}.press-unknown{color:#888}
.rec-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:10px}
.rec-card{border:1px solid var(--line);border-radius:12px;padding:12px;background:#fff}
.rec-best{border-color:#1f8a4c;box-shadow:0 0 0 2px rgba(31,138,76,.12)}
.sn-row{display:grid;grid-template-columns:1.1fr 1fr 1fr 1.1fr auto;gap:8px;align-items:end;margin-bottom:8px}
.sn-row input,.sn-row select{height:38px}
.inline-err{grid-column:1/-1;padding:8px 10px;border-radius:8px;background:var(--red-soft);color:var(--red);font-size:12px;font-weight:800}
.tbl{width:100%;border-collapse:collapse;font-size:13px}
.tbl th,.tbl td{padding:9px 8px;border-bottom:1px solid var(--line);text-align:left;vertical-align:top}
.tbl th{color:var(--mute);font-size:11px;text-transform:uppercase;letter-spacing:.06em}
.db{display:flex;gap:8px;overflow-x:auto;padding-bottom:8px;margin:14px 0 18px}
.dc{flex:0 0 auto;min-width:76px;padding:10px 12px;border:1px solid var(--line);border-radius:10px;background:#FBFAF8;cursor:pointer;text-align:center}
.dc.on{background:var(--red);border-color:var(--red);color:#fff}
.dc .dw{font-size:11px;color:var(--mute)}.dc.on .dw{color:rgba(255,255,255,.72)}
.dc .dd{margin-top:2px;font-size:16px;font-weight:900}
.dc .dv{margin-top:3px;font-size:11px;font-weight:800}.dc .dv.h{color:var(--green)}.dc .dv.n{color:var(--red)}.dc.on .dv{color:#fff}
.sg{display:grid;grid-template-columns:repeat(auto-fill,minmax(126px,1fr));gap:9px}
.queue-live-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:12px}
.queue-live-card{position:relative;overflow:hidden;display:grid;gap:14px;padding:16px;border:1px solid var(--line);border-radius:14px;background:linear-gradient(145deg,#fff 0,#FBFAF8 72%);box-shadow:0 12px 26px rgba(42,35,28,.07)}
.queue-live-card:before{content:"";position:absolute;inset:0 auto 0 0;width:5px;background:var(--red)}
.queue-live-card.open:before{background:var(--green)}
.queue-live-card.closed{background:linear-gradient(145deg,#FFF8F6 0,#FBFAF8 100%)}
.queue-live-top{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:12px;align-items:start}
.queue-live-name{min-width:0}
.queue-live-name b{display:block;color:var(--ink);font-size:15px;line-height:1.25;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.queue-live-name span{display:block;margin-top:4px;color:var(--mute);font-size:11px;font-weight:800}
.queue-status{display:inline-flex;align-items:center;height:24px;padding:0 9px;border-radius:999px;background:#EEE9E4;color:var(--sub);font-size:11px;font-weight:900}
.queue-status.ok{background:var(--green-soft);color:var(--green)}
.queue-status.bad{background:var(--red-soft);color:var(--red)}
.queue-live-main{display:grid;grid-template-columns:minmax(0,1fr) 150px;gap:12px;align-items:end}
.queue-call span{display:block;color:var(--mute);font-size:11px;font-weight:900;letter-spacing:.04em}
.queue-call strong{display:block;margin-top:2px;color:var(--ink);font-size:38px;line-height:1;font-weight:950;letter-spacing:-.04em;font-variant-numeric:tabular-nums}
.queue-call em{font-style:normal;color:var(--red);font-size:18px}
.queue-spark{min-height:50px;padding:8px;border:1px solid var(--line);border-radius:12px;background:#fff}
.queue-metrics{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:8px}
.queue-metric{padding:10px;border-radius:11px;background:#F7F4F1;border:1px solid rgba(222,216,210,.75)}
.queue-metric span{display:block;color:var(--mute);font-size:11px;font-weight:900}
.queue-metric b{display:block;margin-top:4px;color:var(--ink);font-size:17px;font-weight:950;font-variant-numeric:tabular-nums}
.queue-meter{height:8px;border-radius:999px;background:#EDE8E3;overflow:hidden}
.queue-meter i{display:block;height:100%;border-radius:999px;transition:width .3s}
.queue-meter .lv-g{background:#2E9B5B}
.queue-meter .lv-y{background:#C8881A}
.queue-meter .lv-r{background:var(--red)}
.queue-live-foot{display:flex;align-items:center;justify-content:space-between;gap:10px;flex-wrap:wrap}
.queue-live-foot span{color:var(--mute);font-size:11px;font-weight:800;line-height:1.5}
.queue-live-note{margin-top:10px;color:var(--mute);font-size:12px;line-height:1.65}
.store-result-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:12px}
.sl{padding:13px;border:1px solid var(--line);border-radius:10px;background:#F7F4F1}
.sl.av{background:var(--green-soft);border-color:#B9DEC2}
.sl.full{background:var(--red-soft);border-color:#F0B7B9}
.sl.fu{opacity:.52}
.sl .tm{font-size:15px;font-weight:900;color:var(--ink)}
.sl .ss{margin-top:4px;font-size:12px;color:var(--sub)}
.pill{display:inline-flex;align-items:center;height:24px;padding:0 9px;border-radius:999px;background:#EEE9E4;color:var(--sub);font-size:11px;font-weight:900}
.disc-list{display:grid;grid-template-columns:repeat(auto-fill,minmax(300px,1fr));gap:12px}
.disc-list .sl{padding:14px}
.lg{max-height:430px;overflow:auto;padding:14px;border-radius:10px;background:#181614;color:#E8E1DA;font-family:"SF Mono",Menlo,Consolas,monospace;font-size:12px;line-height:1.75}
.ll{display:flex;gap:10px;border-bottom:1px solid rgba(255,255,255,.06);padding:2px 0}
.ll .lt{color:#9F988F;flex:0 0 auto}.ll.er .lm{color:#FFB7B7}
.empty{padding:32px;border:1px dashed var(--line-strong);border-radius:10px;text-align:center;color:var(--mute);background:#FBFAF8}
.errbox{margin-bottom:12px;padding:12px;border:1px solid #F0B7B9;border-radius:10px;background:var(--red-soft);color:var(--red);font-size:13px;line-height:1.6}
.diag-detail{margin-top:12px;padding:14px;border:1px solid var(--line);border-radius:10px;background:#FBFAF8;color:var(--sub);font-size:12px;line-height:1.7}
.diag-detail b{color:var(--ink)}
.diag-detail.bad{border-color:var(--red);background:#FEF6F4}
.spark{width:140px;height:34px;flex:none;opacity:.9}
.spark polyline{fill:none;stroke:var(--red);stroke-width:2;vector-effect:non-scaling-stroke;stroke-linejoin:round;stroke-linecap:round}
.waitbar{height:8px;border-radius:999px;background:#EDE8E3;overflow:hidden;margin:10px 0 2px}
.waitbar i{display:block;height:100%;border-radius:999px;transition:width .3s}
.waitbar .lv-g{background:#2E9B5B}
.waitbar .lv-y{background:#C8881A}
.waitbar .lv-r{background:var(--red)}
.qbox{border:1px solid var(--line);border-radius:12px;padding:14px;background:#FBFAF8}
.pick-out{margin-top:8px;padding:12px 14px;border-radius:10px;background:var(--paper);border:1px solid var(--line);font-size:14px;line-height:1.6}
.pick-out b{color:var(--red);font-size:16px}
.home-decision{padding:18px;border:1px solid var(--line);border-radius:12px;background:linear-gradient(135deg,#fff 0,#FBFAF8 100%);box-shadow:0 8px 24px rgba(42,35,28,.05)}
.home-decision-head{display:flex;align-items:flex-start;justify-content:space-between;gap:14px;margin-bottom:12px}
.home-decision-head h2{margin:6px 0 0;color:var(--ink);font-size:20px;line-height:1.18;letter-spacing:0;font-weight:950}
.home-decision-head p{margin:0;color:var(--sub);font-size:12px;line-height:1.7;max-width:320px}
.home-decision-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:10px}
.home-decision-card{position:relative;display:flex;flex-direction:column;align-items:flex-start;min-height:122px;padding:14px;border:1px solid var(--line);border-radius:10px;background:#fff;color:var(--text);cursor:pointer;text-align:left;box-shadow:0 8px 18px rgba(42,35,28,.04);transition:transform .14s,box-shadow .14s,border-color .14s}
.home-decision-card:hover{transform:translateY(-2px);box-shadow:0 14px 26px rgba(42,35,28,.09);border-color:var(--line-strong)}
.home-decision-card span{display:inline-flex;height:24px;align-items:center;padding:0 9px;border-radius:999px;font-size:11px;font-weight:950}
.home-decision-card.read span{color:var(--green);background:var(--green-soft)}
.home-decision-card.auth span{color:var(--blue);background:var(--blue-soft)}
.home-decision-card b{display:block;margin-top:10px;color:var(--ink);font-size:17px;line-height:1.25}
.home-decision-card small{display:block;margin-top:6px;color:var(--sub);font-size:12px;line-height:1.55}
.task-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px;margin-top:18px}
.task-card{position:relative;overflow:hidden;text-align:left;min-height:164px;padding:18px;border:1px solid var(--line);border-radius:16px;background:linear-gradient(145deg,#fff 0,#FBFAF8 74%);box-shadow:0 10px 24px rgba(42,35,28,.06);cursor:pointer;color:var(--text)}
.task-card:hover{transform:translateY(-2px);box-shadow:0 18px 36px rgba(42,35,28,.11);border-color:var(--line-strong)}
.task-card:before{content:"";position:absolute;inset:auto 16px 16px auto;width:52px;height:52px;border-radius:999px;background:var(--red-soft)}
.task-card h3{position:relative;margin:12px 0 7px;color:var(--ink);font-size:22px;line-height:1.18;letter-spacing:-.02em}
.task-card p{position:relative;color:var(--sub);font-size:13px;line-height:1.65}
.task-foot{position:relative;display:flex;align-items:center;justify-content:space-between;gap:8px;margin-top:14px;flex-wrap:wrap}
.task-arrow{display:inline-flex;align-items:center;justify-content:center;width:30px;height:30px;border-radius:999px;background:var(--ink);color:#fff;font-weight:900}
.tag{display:inline-flex;align-items:center;height:24px;padding:0 9px;border-radius:999px;border:1px solid var(--line-strong);background:#fff;color:var(--sub);font-size:11px;font-weight:950;white-space:nowrap}
.tag.read{color:var(--green);border-color:#BFE4CC;background:var(--green-soft)}
.tag.auth{color:var(--blue);border-color:#B9D2E4;background:var(--blue-soft)}
.tag.action{color:var(--red);border-color:#F0B7B9;background:var(--red-soft)}
.setting-fold{grid-column:1/-1;padding:0;overflow:hidden}
.setting-fold>summary{display:flex;align-items:center;justify-content:space-between;gap:14px;list-style:none;cursor:pointer;padding:18px 20px}
.setting-fold>summary::-webkit-details-marker{display:none}
.setting-fold>summary:hover{background:#FBFAF8}
.setting-fold{transition:box-shadow .15s}
.setting-fold[open]{box-shadow:0 4px 14px rgba(25,24,23,.05)}
.setting-fold-title b{display:block;color:var(--ink);font-size:15px;letter-spacing:.02em}
.setting-fold-title span{display:block;margin-top:5px;color:var(--sub);font-size:12px;line-height:1.55}
.setting-fold>summary:after{content:'展开';display:inline-flex;align-items:center;height:26px;padding:0 10px;border-radius:999px;background:#F0EDEA;color:var(--sub);font-size:11px;font-weight:900;white-space:nowrap}
.setting-fold[open]>summary:after{content:'收起';background:var(--red-soft);color:var(--red)}
.setting-fold-body{border-top:1px solid var(--line);padding:16px 20px 20px}
.page-lead{display:flex;align-items:flex-start;justify-content:space-between;gap:16px;flex-wrap:wrap;margin-bottom:16px}
.page-lead h2{margin:0;color:var(--ink);font-size:30px;line-height:1.1;letter-spacing:-.03em}
.page-lead p{margin-top:8px;color:var(--sub);font-size:13px;line-height:1.75;max-width:720px}
.ph{margin:0;color:var(--ink);font-size:24px;line-height:1.16;letter-spacing:-.02em;font-weight:900}
.ph-sub{margin-top:7px;color:var(--sub);font-size:13px;line-height:1.7;max-width:760px}
.btn-more{border:0;border-top:1px dashed var(--line);margin-top:0}
.btn-more>summary{cursor:pointer;list-style:none;padding:8px 0 0;font-size:12px;font-weight:800;color:var(--mute)}
.btn-more>summary::-webkit-details-marker{display:none}
.btn-more>summary::before{content:'⋯ 更多操作';}
.btn-more[open]>summary::before{content:'▾ 更多操作';}
.btn-more.danger>summary{color:var(--red)}
.btn-more.danger>summary::before{content:'⚠ 危险操作';}
.btn-more.danger[open]>summary::before{content:'▾ 危险操作';}
.btn-more .fl{margin-top:10px}
.quick-panel{display:grid;grid-template-columns:1fr auto;gap:12px;align-items:end;padding:16px;border:1px solid var(--line);border-radius:14px;background:#FBFAF8;margin-bottom:16px}
.quick-panel strong{display:block;color:var(--ink);font-size:18px}
.quick-panel p{margin-top:6px;color:var(--sub);font-size:13px;line-height:1.65}
.adv{border-top:1px dashed var(--line);padding-top:4px}
.adv>summary{cursor:pointer;list-style:none;padding:12px 0;font-weight:800;color:var(--sub);font-size:13px}
.adv>summary::-webkit-details-marker{display:none}
.adv>summary::before{content:'▸ ';color:var(--mute)}
.adv[open]>summary::before{content:'▾ '}
.diag-detail code{display:inline-block;max-width:100%;overflow:auto;padding:2px 5px;border-radius:6px;background:#EEE9E4;color:var(--ink)}
.ft{padding:26px 0 46px;text-align:center;color:var(--mute);font-size:12px}.ft a{color:var(--red);text-decoration:none}
.hid{display:none!important}.mu{color:var(--mute)}.tc{text-align:center}.tg{color:var(--green)}.tre{color:var(--red)}
.mt8{margin-top:8px}.mt16{margin-top:16px}.mb16{margin-bottom:16px}
.fl{display:flex}.g8{gap:8px}.g12{gap:12px}.ai{align-items:center}.jb{justify-content:space-between}.fw{flex-wrap:wrap}
@media(max-width:900px){
  .grid,.settings-grid,.sn-row,.dash-split,.dash-head,.advisor-card,.task-grid,.quick-panel,.curve-sampling,.home-decision-grid{grid-template-columns:1fr}
  .dash-controls{justify-content:flex-start}
  .curve-sampling-actions{justify-content:flex-start}
  .queue-live-main{grid-template-columns:1fr}
  .queue-metrics{grid-template-columns:1fr}
  .hdr-in{height:auto;min-height:70px;flex-wrap:wrap;padding:12px 0}
  .nav{order:3;width:100%;overflow:auto}
  .nav a{flex:1;justify-content:center}
  .ver{margin-left:auto}
}
@media(max-width:600px){
  .shell{padding:0 14px}.wrap{padding-top:18px}
  .hdr-in{min-height:58px;padding:8px 0;gap:10px}
  .brand img{width:36px;height:36px}.brand strong{font-size:15px}.brand span{font-size:10px}
  .nav{padding:4px}.nav a{height:32px;padding:0 12px}
  .hero{min-height:auto;padding:18px}.hero h1{margin-top:14px;font-size:25px;line-height:1.18}.hero p{margin-top:8px;font-size:13px;line-height:1.65}
  .hero-pm{display:none}.actions{margin-top:16px}.bt-l{height:42px;padding:0 18px;font-size:14px}
  .hero-pick{margin-top:10px}.hero-pick .bt{height:42px;padding:0 16px;font-size:13px}.hero-pick span{font-size:12px;line-height:1.55}
  .home-decision{padding:14px}.home-decision-head{display:block}.home-decision-head p{margin-top:8px;max-width:none}.home-decision-card{min-height:auto}
  .actions .bt{width:100%}.side{gap:10px}
  .card,.cd{padding:16px}
  .ovc{padding:14px}
  .dash-chart,.chart{overflow:hidden}
  .dash-chart svg,.chart svg{min-width:0;height:auto}
  #qdPressChart{overflow:auto}
  #qdPressChart svg{min-width:0;height:auto}
}
/* 中等宽度（平板竖屏 / 大手机 600-768px）：多列网格降为 2 列，避免拥挤 */
@media(min-width:601px) and (max-width:768px){
  .home-decision-grid,.journey-steps,.mechanism-map,.task-grid{grid-template-columns:repeat(2,minmax(0,1fr))}
  .sn-row{grid-template-columns:1fr 1fr;gap:8px}
  .advisor-card{grid-template-columns:1fr}
}
.ov{position:fixed;inset:0;z-index:50;background:rgba(25,24,23,.45);display:flex;align-items:center;justify-content:center;padding:18px}
.ov.hid{display:none}
.ovc{width:min(680px,96vw);max-height:88vh;display:flex;flex-direction:column;background:var(--paper);border:1px solid var(--line);border-radius:16px;padding:18px;box-shadow:0 24px 60px rgba(25,24,23,.28)}
.splist{flex:1;overflow:auto;border:1px solid var(--line);border-radius:12px;padding:8px;background:#FBFAF8}
.spgroup{margin-bottom:10px}
.spcity{font-size:12px;font-weight:900;color:var(--sub);padding:6px 4px;position:sticky;top:-8px;background:#FBFAF8}
.sprow{display:flex;align-items:center;gap:10px;padding:8px 10px;border-radius:10px;cursor:pointer}
.sprow:hover{background:#fff}
.sprow.on{background:var(--red-soft)}
.sprow input{width:auto;height:auto;flex:none}
.spname{flex:1;font-size:14px;font-weight:800;color:var(--ink)}
.spname .mu{font-weight:600}
.spbs{display:flex;gap:5px;flex-wrap:wrap;justify-content:flex-end}
.spb{font-size:11px;font-weight:800;padding:2px 8px;border-radius:999px;border:1px solid var(--line-strong);color:var(--sub);background:#fff}
.spb.ok{color:#1A7F47;border-color:#BFE4CC;background:#F0FAF3}
.spb.warn{color:#9A6700;border-color:#F0DDA8;background:#FFF8E8}
.spb.mut{color:var(--mute)}
.toast-wrap{position:fixed;right:18px;bottom:18px;z-index:80;display:flex;flex-direction:column;gap:8px;align-items:flex-end;pointer-events:none}
.toast{pointer-events:auto;max-width:min(380px,86vw);padding:12px 14px;border-radius:12px;background:var(--ink);color:#fff;font-size:13px;font-weight:700;line-height:1.5;box-shadow:0 14px 34px rgba(25,24,23,.26);border-left:4px solid var(--mute);opacity:0;transform:translateY(10px);transition:opacity .22s,transform .22s;white-space:pre-line}
.toast.in{opacity:1;transform:translateY(0)}
.toast.ok{border-left-color:var(--green)}
.toast.err{border-left-color:var(--red)}
.toast.warn{border-left-color:var(--yellow)}
.toast.info{border-left-color:var(--blue)}
.confirm-ovc{width:min(440px,94vw)}
.confirm-h{display:flex;align-items:center;gap:8px;font-size:17px;font-weight:900;color:var(--ink)}
.confirm-b{margin-top:10px;color:var(--sub);font-size:13.5px;line-height:1.7;white-space:pre-line}
.confirm-acts{display:flex;justify-content:flex-end;gap:10px;margin-top:18px}
.confirm-danger .confirm-h{color:var(--red)}
@media(max-width:600px){.toast-wrap{right:10px;left:10px;bottom:10px;align-items:stretch}.toast{max-width:none}}
.kpi-hot{border-color:rgba(184,28,34,.32);background:linear-gradient(160deg,#fff 0,var(--red-soft) 100%)}
.kpi-hot strong{color:var(--red)}
.skeleton{position:relative;overflow:hidden;background:#EFEBE7}
.skeleton::after{content:"";position:absolute;inset:0;transform:translateX(-100%);background:linear-gradient(90deg,transparent 0,rgba(255,255,255,.6) 50%,transparent 100%);animation:shimmer 1.25s infinite}
.skk{height:84px;border-radius:12px}
@keyframes shimmer{100%{transform:translateX(100%)}}
@media(max-width:600px){
  .called-table{min-width:0}
  .called-table thead{display:none}
  .called-table,.called-table tbody,.called-table tr,.called-table td{display:block;width:auto}
  .called-table tr{border:1px solid var(--line);border-radius:10px;margin-bottom:8px;padding:4px 6px;background:#fff}
  .called-table td{display:flex;align-items:baseline;justify-content:space-between;gap:12px;border:0;padding:6px 8px;text-align:right}
  .called-table td::before{content:attr(data-label);color:var(--mute);font-weight:800;font-size:11px;text-align:left;white-space:nowrap}
  .tbl-cards thead{display:none}
  .tbl-cards,.tbl-cards tbody,.tbl-cards tr,.tbl-cards td{display:block;width:auto}
  .tbl-cards tr{border:1px solid var(--line);border-radius:10px;margin-bottom:8px;padding:4px 6px;background:#fff}
  .tbl-cards td{display:flex;align-items:baseline;justify-content:space-between;gap:12px;border:0;padding:5px 6px;text-align:right}
  .tbl-cards td::before{content:attr(data-label);color:var(--mute);font-weight:800;font-size:11px;text-align:left;white-space:nowrap}
}
.authpill{margin-left:6px;display:inline-flex;align-items:center;height:26px;padding:0 10px;border-radius:999px;font-size:11px;font-weight:800;cursor:pointer;border:1px solid var(--line-strong);background:#fff;color:var(--sub);white-space:nowrap}
.authpill.ok{color:var(--green);border-color:#BFE4CC;background:var(--green-soft)}
.authpill.stale{color:var(--red);border-color:#F0B7B9;background:var(--red-soft)}
.authbanner{display:flex;align-items:center;gap:10px;flex-wrap:wrap;margin:0 0 16px;padding:12px 14px;border-radius:12px;background:var(--red-soft);border:1px solid #F0B7B9;color:var(--red);font-size:13px;font-weight:700;cursor:pointer}
.authbanner b{font-weight:900;margin-right:2px}
.authbanner .bt{margin-left:auto}
/* v3 体验改造：三态首页 / 通行证向导 / 健康胶囊（docs/ux-redesign-v3.md） */
.hero-pick{display:flex;flex-direction:column;gap:8px;align-items:flex-start;margin-top:14px}
.ticket-hero{position:relative;margin-bottom:16px;padding:22px 24px;border-radius:18px;background:linear-gradient(135deg,var(--red) 0%,var(--red-dark) 100%);color:#fff;box-shadow:0 10px 28px rgba(184,28,34,.22)}
.ticket-hero .th-eyebrow{font-size:12px;font-weight:800;letter-spacing:.08em;opacity:.85}
.ticket-hero .th-no{font-size:34px;font-weight:900;letter-spacing:.02em;margin:4px 0 2px}
.ticket-hero .th-line{font-size:14px;font-weight:700;opacity:.95;margin-top:6px;line-height:1.7}
.ticket-hero .th-sub{font-size:12px;opacity:.8;margin-top:2px}
.ticket-hero .th-acts{display:flex;gap:8px;flex-wrap:wrap;margin-top:14px;align-items:center}
.ticket-hero .bt-w{background:#fff;color:var(--red);border-color:#fff}
.ticket-hero .bt-ghost{background:transparent;color:#fff;border:1px solid rgba(255,255,255,.55)}
.wsteps{display:flex;align-items:flex-start;margin:4px 0 18px}
.wstep{flex:1;display:flex;flex-direction:column;align-items:center;gap:6px;position:relative;font-size:11px;font-weight:800;color:var(--mute)}
.wstep i{width:22px;height:22px;border-radius:50%;border:2px solid var(--line-strong);background:#fff;display:flex;align-items:center;justify-content:center;font-style:normal;font-size:11px;color:var(--mute);z-index:1}
.wstep.done i{background:var(--green);border-color:var(--green);color:#fff}
.wstep.on i{background:var(--red);border-color:var(--red);color:#fff}
.wstep.on{color:var(--ink)}
.wstep.done{color:var(--green)}
.wstep::before{content:'';position:absolute;top:11px;left:-50%;width:100%;height:2px;background:var(--line)}
.wstep:first-child::before{display:none}
.wnum{display:flex;gap:10px;align-items:flex-start;padding:10px 12px;border:1px solid var(--line);border-radius:12px;margin-top:8px;background:#FBFAF8;line-height:1.7;font-size:13px}
.wnum b.n{flex:none;width:22px;height:22px;border-radius:50%;background:var(--red-soft);color:var(--red);display:flex;align-items:center;justify-content:center;font-size:12px}
.why{margin-top:10px;padding:8px 12px;border-radius:10px;background:var(--yellow-soft);color:var(--yellow);font-size:12px;font-weight:700;line-height:1.7}
.mascot-wrap{text-align:center;padding:10px 0}
.mascot-row{display:flex;flex-wrap:wrap;justify-content:center;align-items:flex-end;gap:10px 12px}
.mascot-row .mascot:nth-child(2n){transform:translateY(5px)}
.mascot-row .mascot{transition:transform .25s}
.mascot-row .mascot:hover{transform:translateY(-4px)}
.celebrate{position:relative;overflow:hidden;text-align:center;padding:18px 0 8px}
.confetti{position:absolute;top:-12px;width:8px;height:12px;border-radius:2px;opacity:.9;animation:cfall 1.4s ease-in forwards}
@keyframes cfall{to{transform:translateY(240px) rotate(540deg);opacity:0}}
.strip{display:flex;align-items:center;gap:10px;padding:10px 12px;border:1px solid var(--line);border-radius:12px;margin-top:8px}
.strip>div{flex:1}
.strip b{display:block;font-size:13px}
.strip span.sd{display:block;font-size:11px;color:var(--mute);margin-top:2px}
.strip .st{flex:none;font-size:11px;font-weight:800;padding:2px 8px;border-radius:999px}
.strip .st.ok{color:var(--green);background:var(--green-soft)}
.strip .st.warn{color:var(--yellow);background:var(--yellow-soft)}
.strip .st.bad{color:var(--red);background:var(--red-soft)}
.authpill.warn{color:var(--yellow);border-color:#EBD9A8;background:var(--yellow-soft)}
.journey-panel{padding:16px;border:1px solid var(--line);border-radius:14px;background:linear-gradient(135deg,#fff 0,#FBFAF8 100%);box-shadow:0 10px 24px rgba(42,35,28,.05)}
.journey-panel.warn{border-color:#E8D7AA;background:linear-gradient(135deg,#FFFDF7 0,#FBFAF8 100%)}
.journey-panel.bad{border-color:#F0B7B9;background:linear-gradient(135deg,#FFF6F6 0,#FBFAF8 100%)}
.journey-head{display:flex;align-items:flex-start;justify-content:space-between;gap:12px;flex-wrap:wrap}
.journey-kicker{display:inline-flex;align-items:center;height:24px;padding:0 9px;border-radius:999px;background:#EEE9E4;color:var(--sub);font-size:11px;font-weight:950}
.journey-head h2{margin:9px 0 0;color:var(--ink);font-size:22px;line-height:1.2;font-weight:950}
.journey-copy{margin-top:7px;color:var(--sub);font-size:13px;line-height:1.65;max-width:700px}
.journey-mode{display:inline-flex;align-items:center;min-height:28px;padding:0 10px;border-radius:999px;border:1px solid var(--line-strong);background:#fff;color:var(--sub);font-size:12px;font-weight:900;white-space:nowrap}
.journey-mode.ok{color:var(--green);border-color:#BFE4CC;background:var(--green-soft)}
.journey-mode.warn{color:var(--yellow);border-color:#EBD9A8;background:var(--yellow-soft)}
.journey-mode.bad{color:var(--red);border-color:#F0B7B9;background:var(--red-soft)}
.journey-steps{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:8px;margin-top:13px}
.journey-step{min-height:96px;padding:11px 12px;border:1px solid var(--line);border-radius:12px;background:#fff}
.journey-step span{display:inline-flex;height:22px;align-items:center;padding:0 8px;border-radius:999px;font-size:11px;font-weight:950;background:#EEE9E4;color:var(--sub)}
.journey-step b{display:block;margin-top:8px;color:var(--ink);font-size:14px;line-height:1.3}
.journey-step small{display:block;margin-top:4px;color:var(--sub);font-size:11px;line-height:1.5}
.journey-step.read span,.journey-step.ok span{color:var(--green);background:var(--green-soft)}
.journey-step.auth span,.journey-step.warn span{color:var(--yellow);background:var(--yellow-soft)}
.journey-step.action span,.journey-step.bad span{color:var(--red);background:var(--red-soft)}
.journey-cta{display:flex;gap:8px;flex-wrap:wrap;margin-top:13px}
.mechanism-map{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:8px}
.mechanism-item{min-height:118px;padding:12px;border:1px solid var(--line);border-radius:12px;background:#fff}
.mechanism-item span{display:inline-flex;height:22px;align-items:center;padding:0 8px;border-radius:999px;background:#EEE9E4;color:var(--sub);font-size:11px;font-weight:950}
.mechanism-item.read span{color:var(--green);background:var(--green-soft)}
.mechanism-item.auth span{color:var(--yellow);background:var(--yellow-soft)}
.mechanism-item.action span{color:var(--red);background:var(--red-soft)}
.mechanism-item b{display:block;margin-top:8px;color:var(--ink);font-size:15px;line-height:1.3}
.mechanism-item p{margin-top:5px;color:var(--sub);font-size:12px;line-height:1.6}
.home-live{display:grid;grid-template-columns:repeat(auto-fit,minmax(190px,1fr));gap:12px}
.hl-card{display:flex;flex-direction:column;align-items:flex-start;gap:2px;padding:14px 16px;border:1px solid var(--line);border-radius:14px;background:var(--paper);cursor:pointer;text-align:left;transition:transform .15s,box-shadow .15s}
.hl-card:hover{transform:translateY(-2px);box-shadow:0 6px 16px rgba(25,24,23,.08)}
.hl-name{font-size:12px;font-weight:800;color:var(--sub)}
.hl-num{font-size:30px;font-weight:900;color:var(--red);line-height:1.15}
.hl-num.closed{color:var(--mute)}
.hl-sub{font-size:11px;color:var(--mute)}
.diag-next{margin-bottom:12px;padding:15px 16px;border:1px solid var(--line);border-radius:12px;background:#fff}
.diag-next h3{margin:0;color:var(--ink);font-size:18px;line-height:1.25}
.diag-next p{margin-top:7px;color:var(--sub);font-size:13px;line-height:1.65}
.diag-next.ok{border-color:#BFE4CC;background:var(--green-soft)}
.diag-next.warn{border-color:#EBD9A8;background:var(--yellow-soft)}
.diag-next.bad{border-color:#F0B7B9;background:var(--red-soft)}
@media(max-width:900px){.journey-steps,.mechanism-map{grid-template-columns:1fr}}
.pm{display:inline-flex;vertical-align:middle}
.hero-pm{position:absolute;top:14px;right:16px;opacity:.95}
.hero{position:relative}
.belt{overflow:hidden;margin-top:36px;border-top:2px solid var(--line);border-bottom:2px solid var(--line);background:repeating-linear-gradient(90deg,#F4F1EE 0 46px,#ECE7E2 46px 92px);height:64px}
.belt-track{display:flex;width:max-content;padding-top:8px;animation:beltmove 60s linear infinite}
.belt-item{display:flex;flex-direction:column;align-items:center;margin-right:56px}
.belt-item .plate{width:48px;height:9px;border-radius:50%;background:#fff;border:2px solid #E0DAD4;margin-top:-7px;box-shadow:0 2px 3px rgba(0,0,0,.08)}
.belt-item:nth-child(4n+2) .plate{border-color:#F5BA24}
.belt-item:nth-child(4n+3) .plate{border-color:#B81C22}
.belt-item:nth-child(4n) .plate{border-color:#9B9691}
@keyframes beltmove{to{transform:translateX(-50%)}}
@media (prefers-reduced-motion:reduce){.belt-track{animation:none}}
/* ===== 溢出兜底 + 组件统一（修 qd/qt 文字越界、输入框高度不一）=====
   根因：多处 flex/grid 子项没有 min-width:0，长数字/长文本把容器撑爆；
   部分大字号（叫号 38px、KPI 28px）在窄列里溢出。这里统一加溢出兜底。 */
.grid,.settings-grid,.dash-split,.quick-panel,.advisor-card,.curve-sampling,.queue-live-top,.queue-live-main,.queue-live-foot,.home-decision-grid,.journey-head,.rank-row,.store-row,.sprow,.strip,.fl{min-width:0}
/* 卡片本身不撑破父级 */
.cd,.card,.qbox,.answer-card,.rec-card,.queue-live-card,.kpi,.hl-card,.ci,.metric,.sample-state,.data-source,.pick-out,.diag-next,.status-card,.home-decision{min-width:0;overflow-wrap:anywhere;word-break:break-word}
/* 叫号大数字：允许收缩，过长（如老号码上千）按字号自适应，不顶破右侧折线 */
.queue-call{min-width:0}
.queue-call strong{font-size:clamp(26px,7vw,38px);letter-spacing:-.02em;overflow-wrap:anywhere}
.queue-call em{font-size:.5em;vertical-align:middle}
/* KPI / 指标大数字不溢出 */
.kpi strong,.advisor-point strong{overflow-wrap:anywhere;hyphens:auto}
.queue-metric b,.queue-metric span{min-width:0;overflow-wrap:anywhere}
/* 胶囊 / 标签：内容过长换行而非顶破；高度保留单行时的视觉 */
.ci{overflow-wrap:anywhere;word-break:break-word}
.chip,.tag,.pill,.queue-status,.spb,.journey-mode,.journey-kicker,.advisor-eyebrow,.authpill,.dash-target{max-width:100%;overflow-wrap:anywhere;word-break:break-word}
/* answer-chip 里的 strong（数字+单位）允许换行 */
.answer-chip{min-width:0}.answer-chip strong{overflow-wrap:anywhere}
/* 网格列在窄屏不强行保持原始 minmax，避免横向溢出 */
.queue-metrics,.metric,.kpi-grid,.cg,.sample-state,.rec-grid,.home-decision-grid,.queue-live-grid,.store-result-grid,.disc-list{min-width:0}
/* 趋势/热力/表格的水平滚动容器已有 overflow:auto，确保其父级不撑破 */
.dash-chart,.chart,.heat-wrap,.db,.called-table-wrap{min-width:0}
/* 标题/副标题在窄屏换行而非溢出 */
.dash-title,.dash-copy,.ph,.ph-sub,.page-lead h2,.page-lead p,.hero h1,.hero p{overflow-wrap:anywhere}
/* 头部品牌/版本在窄屏不撑破 */
.brand{min-width:0;overflow:hidden}
.brand strong,.brand span{overflow-wrap:anywhere;max-width:100%}
.nav{min-width:0}
.nav a{overflow-wrap:anywhere}
/* ===== 组件统一：输入框高度统一为 40px，胶囊与同行对齐 ===== */
/* 输入框统一 40px（.dash-target 的内嵌 input 保持内陷样式，不顶满容器） */
.tr input,.sn-row input,.sn-row select{height:40px;border-radius:8px}
.dash-target{height:40px}
.dash-target input{height:28px;border:0;background:#F7F4F1;border-radius:999px}
.dash-target input:focus{outline:0;border:0;box-shadow:none}
.dash-datepick{display:flex;align-items:center;gap:7px;height:40px;padding:0 12px;border:1px solid var(--line-strong);border-radius:999px;background:#fff;color:var(--sub);font-size:12px;font-weight:900;max-width:100%}
.dash-datepick select{border:0;background:transparent;font-size:12px;font-weight:900;color:var(--ink);cursor:pointer;max-width:9rem}
/* 表格内紧凑输入保持 82px 宽，高度对齐 */
.tr input{width:82px;text-align:center}
/* select 统一（全局已是 40px，这里补齐 .fr/.fg 内嵌的） */
.fr .fg select,.fr .fg input{min-width:0}
/* 胶囊型组件与 40px 输入对齐；纯按钮 .bt-s 保留 34px 不动，靠 align-items:center 居中 */
.switch,.check,.chip{min-height:38px;border-radius:999px}
.switch input,.check input{margin:0}
.preset{min-height:42px;border-radius:10px}
/* disabled 输入样式 */
input:disabled,select:disabled,textarea:disabled{background:#F3F0ED;color:var(--mute);cursor:not-allowed;border-color:var(--line)}
input:disabled:focus,select:disabled:focus,textarea:disabled:focus{box-shadow:none;border-color:var(--line)}
/* tooltip 始终在 sticky header 之上、overlay 之下 */
.dash-tip{z-index:30}
/* 屏幕自适应：中等/窄屏下控制条、表单行、换算行收敛 */
@media(max-width:900px){
  .dash-controls{flex-wrap:wrap;gap:8px}
  .dash-controls .dash-target,.dash-controls .dash-datepick{flex:1 1 auto;min-width:0}
  .dash-controls .bt{flex:0 0 auto}
  .fr{gap:10px}
  .plan-row{flex-direction:column;align-items:stretch}
  .plan-row .fg{width:100%}
}
@media(max-width:600px){
  .fr{flex-direction:column;align-items:stretch}
  .fr .fg{width:100%}
  .fr .fg[style*="align-self"]{align-self:auto!important}
  .dash-head{grid-template-columns:1fr}
  .dash-title{font-size:22px}
  .hero h1{font-size:23px}
  .plan-head{flex-direction:column}
  .plan-swap{align-self:flex-start}
}
/* 时间换算器（几点取号 ⇄ 几点吃）—— 一等公民卡片 */
.plan-card{border:1px solid var(--line);border-radius:14px;padding:18px;background:linear-gradient(135deg,#fff 0,#FBFAF8 100%);box-shadow:0 8px 24px rgba(42,35,28,.05)}
.plan-card:before{content:"";display:block;height:4px;margin:-18px -18px 14px;border-radius:14px 14px 0 0;background:linear-gradient(90deg,var(--red),#F5BA24)}
.plan-head{display:flex;align-items:flex-start;justify-content:space-between;gap:12px}
.plan-head b{display:block;color:var(--ink);font-size:17px}
.plan-head .mu{margin-top:5px;font-size:12px;line-height:1.6;max-width:560px}
.plan-swap{flex:none;display:inline-flex;align-items:center;justify-content:center;width:42px;height:42px;border-radius:50%;border:1px solid var(--line-strong);background:#fff;color:var(--red);font-size:20px;font-weight:900;cursor:pointer;transition:transform .18s,background .18s}
.plan-swap:hover{transform:rotate(180deg);background:var(--red-soft)}
.plan-row{display:flex;gap:12px;flex-wrap:wrap;margin-top:14px}
.plan-basis>summary{cursor:pointer;list-style:none;font-size:12px;font-weight:800;color:var(--mute);padding-top:6px}
.plan-basis>summary::-webkit-details-marker{display:none}
.plan-basis>summary::before{content:'▸ 为什么这么算 '}
.plan-basis[open]>summary::before{content:'▾ '}
/* 设置页危险操作隔离区（NN/G proximity：与良性配置空间隔离） */
.danger-zone{margin-top:24px;padding:16px 18px;border:1.5px solid #F0B7B9;border-radius:14px;background:linear-gradient(135deg,#FFF6F6 0,#fff 60%)}
.danger-zone-head{display:flex;align-items:flex-start;justify-content:space-between;gap:12px;flex-wrap:wrap}
.danger-zone-head b{color:var(--red);font-size:15px}
/* 「我有号码」提醒卡输入框美化（qdr/nr 系列） */
#qdReminderCard .fr{gap:12px}
#qdReminderCard .fg label{font-size:11px;font-weight:800;color:var(--mute);letter-spacing:.02em}
#qdReminderCard select,#qdReminderCard input[type=text],#qdReminderCard input[type=number],#qdReminderCard input[type=time]{height:40px;border-radius:10px;border:1px solid var(--line-strong);background:#fff;transition:border-color .14s,box-shadow .14s}
#qdReminderCard select:focus,#qdReminderCard input:focus{border-color:var(--red);box-shadow:0 0 0 3px rgba(184,28,34,.08);outline:0}
#qdReminderCard .chip{min-height:38px}
#qdReminderCard>div>b{font-size:15px}
/* 设置页分区标题（把「垃圾桶式」平铺折叠归类为有层级的分区） */
.sect-divider{grid-column:1/-1;display:flex;align-items:center;gap:10px;margin:22px 0 2px;padding-top:18px;border-top:1px solid var(--line)}
.sect-divider:first-of-type{margin-top:8px;border-top:0;padding-top:0}
.sect-divider .sect-no{flex:none;display:inline-flex;align-items:center;justify-content:center;width:26px;height:26px;border-radius:8px;background:var(--ink);color:#fff;font-size:13px;font-weight:900}
.sect-divider b{color:var(--ink);font-size:15px;letter-spacing:.01em}
.sect-divider .mu{font-size:12px;margin-left:auto;text-align:right;max-width:60%}
</style>
</head>
<body>
<div class="topline"></div>
<header class="hdr">
  <div class="shell hdr-in">
    <div class="brand">
      <img src="data:image/png;base64,` + logoBase64 + `" alt="SUSHIRO">
      <div><strong>SUSHIRO Overdose</strong><span>reservation assistant</span></div>
    </div>
    <nav class="nav top">
      <a href="#" class="on" data-group="home" onclick="goGroup('home');return false">首页</a>
      <a href="#" data-group="eat" onclick="goGroup('eat');return false">现在去吃</a>
      <a href="#" data-group="number" onclick="goGroup('number');return false">我有号码</a>
      <a href="#" data-group="book" onclick="goGroup('book');return false">约未来</a>
      <a href="#" data-group="mine" onclick="goGroup('mine');return false">我的单据</a>
      <a href="#" data-group="settings" onclick="goGroup('settings');return false">设置</a>
    </nav>
    <div class="mode-switch" id="uiModeSwitch" aria-label="界面模式">
      <button type="button" id="uiModeSimple" onclick="setUIMode('simple')">简化</button>
      <button type="button" id="uiModeAdvanced" onclick="setUIMode('advanced')">进阶</button>
    </div>
    <span class="authpill hid" id="authPill" onclick="authPillClick()"></span>
    <span class="ver hid" id="ver"></span>
  </div>
</header>

<main class="shell wrap">
  <div id="authBanner" class="authbanner hid" onclick="startAuth()"></div>
  <nav class="nav subnav hid" id="subnav"></nav>
  <section id="p-da">
    <div class="grid">
      <div>
        <div id="activeHome"></div>
        <div class="hero" id="heroBox">
          <span class="pm hero-pm" data-kind="salmon" data-size="56"></span>
          <span class="pm hero-pm" style="right:80px;top:28px;opacity:.6" data-kind="uni" data-size="40"></span>
          <span class="pm hero-pm" style="right:126px;top:40px;opacity:.35" data-kind="tako" data-size="30"></span>
          <div class="eyebrow" id="heroBadge">到店助手</div>
          <h1 id="heroTitle"><span class="skeleton" style="display:inline-block;width:56%;min-width:220px;height:30px;border-radius:8px"></span></h1>
          <p id="heroCopy"><span class="skeleton" style="display:inline-block;width:82%;height:14px;border-radius:6px"></span></p>
          <div class="actions">
            <button class="bt bt-r bt-l hid" id="bm" onclick="mA()">开始</button>
            <button class="bt bt-o hid" id="bs" onclick="sE()">停止</button>
            <button class="bt bt-w hid" id="bc" onclick="startAuth()">拿通行证</button>
          </div>
          <div id="heroPick" class="hero-pick hid">
            <button class="bt bt-r bt-l" onclick="openGuestStorePicker()">🔍 选一家常去的门店，马上看排队</button>
            <span class="mu">不用登录、不用通行证，10 秒出结果。选过的店会记住，以后各页面自动带入。</span>
          </div>
          <div id="nc" class="notice hid"></div>
        </div>
        <div id="homeDecisionPanel" class="home-decision mt16">
          <div class="home-decision-head">
            <div><div class="eyebrow">寿司郎机制</div><h2>你现在是哪种情况</h2></div>
            <p>通行证只在提交动作前需要；只看排队、叫号预测和图表不用登录。</p>
          </div>
          <div class="home-decision-grid">
            <button class="home-decision-card read" onclick="go('qt')" type="button">
              <span>今天去吃</span>
              <b>先看哪家排得少</b>
              <small>比较门店、等位和当前叫号；远程取当天排队号会再次确认。</small>
            </button>
            <button class="home-decision-card read" onclick="go('qd')" type="button">
              <span>我有当天排队号</span>
              <b>算几点叫到我</b>
              <small>输入号码和门店，估算到店时间；图表直接展开解释依据。</small>
            </button>
            <button class="home-decision-card auth" onclick="currentUIMode()==='advanced'?go('ca'):enterAdvanced('ca')" type="button">
              <span>想约未来某天</span>
              <b>查未来预约</b>
              <small>看未来可约日历；需要创建预约时再获取通行证。</small>
            </button>
          </div>
        </div>
        <div id="journeyPanel" class="journey-panel mt16"><div class="ci">正在判断今天该走哪条路…</div></div>
        <div id="homeLive" class="mt16"></div>
        <div id="cb" class="card hid mt16">
          <h2>通行证捕获进度</h2>
          <div id="cg" class="cg"></div>
        </div>
      </div>
      <aside class="side">
        <div id="eb" class="engine idle"><div class="row"><span class="dot"></span><strong>就绪</strong></div><p>等待操作。</p></div>
        <div class="card hid" id="updBox"></div>
        <div class="card" id="setupCard">
          <h2>准备清单</h2>
          <div id="setupList"></div>
          <div class="fl g8 fw mt16"><button class="bt bt-r bt-s" onclick="openFirstUseWizard()">打开新手引导</button><button class="bt bt-w bt-s" onclick="go('qt')">先看排队</button></div>
        </div>
        <details class="card adv">
          <summary>当前偏好（人数 / 桌型 / 时段）</summary>
          <div class="ps mt16" id="ps"></div>
          <div class="fl g8 fw mt16"><button class="bt bt-w bt-s" onclick="openSnPrefs()">改预约/取号偏好</button><button class="bt bt-w bt-s" onclick="enterAdvanced('re')">我的单据</button></div>
        </details>
      </aside>
    </div>
  </section>

  <section id="p-ca" class="hid advanced-page">
    <div class="cd">
      <div class="fl ai jb mb16 fw g8">
        <div><h2 class="ph">约未来 <span class="pm" data-kind="ikura" data-size="32"></span></h2><p class="ph-sub">这里看的是未来预约，不是当天排队号。查看可约日历和创建预约都需要通行证；只看今天排队请去「现在去吃」。</p></div>
        <div class="fl g8 fw"><button class="bt bt-w bt-s" onclick="saveCalendarStoresAsPrefs()">保存为预约/取号门店</button><button class="bt bt-w bt-s" onclick="rC()">刷新</button><select id="ar" onchange="setAR()" style="width:auto"><option value="0">不自动刷新</option><option value="15">15 秒</option><option value="30">30 秒</option><option value="60">60 秒</option></select></div>
      </div>
      <div class="fg"><label>门店</label><div id="storeChoices" class="chips"><span class="mu">加载中</span></div></div>
      <div class="fl g8 fw mb16">
        <label class="check"><input type="checkbox" id="avOnly" onchange="rC()">只看可预约</label>
        <select id="period" onchange="rC()" style="width:auto"><option value="all">全部时段</option><option value="lunch">午餐</option><option value="dinner">晚餐</option></select>
      </div>
      <div class="db" id="dbar"></div>
      <div id="sc"><div class="empty"><div class="mascot-wrap"><span class="pm" data-kind="plain" data-size="56"></span></div>选择门店查看时段<div class="mt8"><button class="bt bt-r bt-s" onclick="openStorePicker({selected:selStores,onConfirm:applyCalendarStores})">选择门店</button></div></div></div>
    </div>
  </section>

  <section id="p-qd" class="hid">
    <div class="cd">
      <div class="dash-head">
        <div>
          <div class="cd-t" style="margin-bottom:8px">我有号码 <span class="pm" data-kind="maguro" data-size="30"></span></div>
          <div class="dash-title">输入当天排队号，判断几点到店</div>
          <p class="dash-copy"><span class="tag read">只读 · 直接用</span> 这里填的是当天排队号，不是未来预约号。选门店、输入号码后，系统会估算大概几点叫到和建议到店时间；图表直接展开解释依据。</p>
        </div>
        <div id="qdPrimaryActions" class="dash-controls">
          <label class="dash-target">当天排队号 <input id="qdTargetNo" type="number" min="1" placeholder="如 893" oninput="qdInputDebounced()" onkeydown="if(event.key==='Enter'){clearTimeout(_qdInputTimer);loadQueueDashboard()}"></label>
          <label class="dash-datepick">参考曲线 <select id="qdDateType" onchange="loadQueueDashboard()"><option value="all">全部日期</option><option value="weekday">工作日</option><option value="weekend">周末（含周五晚）</option><option value="holiday">节假日</option></select></label>
          <button class="bt bt-r bt-s" onclick="openStorePicker({selected:qdSelected.slice(0,1),multi:false,onConfirm:applyDashboardStores})">选门店</button>
          <button class="bt bt-w bt-s" onclick="loadQueueDashboard()">刷新</button>
        </div>
      </div>
      <div id="qdStores" class="chips mb16"><span class="mu">默认自动选择本机样本最多的门店</span></div>
      <div id="qdPlanFold" class="plan-card mt16">
        <div class="plan-head">
          <div><b id="planTitle">几点取号 → 几点能吃上</b><p class="mu" id="planSub">填取号时间，算出大概几点能吃上</p></div>
          <button type="button" class="plan-swap" id="planSwapBtn" onclick="swapPlanDir()" title="切换计算方向" aria-label="切换计算方向">⇄</button>
        </div>
        <div class="plan-row">
          <div class="fg" id="qpPickupWrap">
            <label>计划取号时间</label>
            <input id="qpPickup" type="time" value="12:10" oninput="runPlanCalcDebounced()">
          </div>
          <div class="fg hid" id="qwMealWrap">
            <label>想几点吃上</label>
            <input id="qwMeal" type="time" value="13:00" oninput="runPlanCalcDebounced()">
          </div>
          <div class="fg hid" id="qwTravelWrap">
            <label>路上几分钟（可选）</label>
            <input id="qwTravel" type="number" min="0" placeholder="如 25" oninput="runPlanCalcDebounced()">
          </div>
        </div>
        <div id="qpAnswer" class="answer-card mt8"><div class="ci">先在上方选门店，再填一个时间，这里立刻给出结果。</div></div>
      </div>
      <div class="cd-t" style="margin-bottom:8px">现在 · 实时排队压力</div>
      <div id="qdAnswer" class="answer-card"><div class="ci">选门店、填号码，这里按当前实时叫号速度告诉你大概几点叫到、几点出发。</div></div>
      <div class="cd-t" style="margin:16px 0 8px">历史规律 · 到店建议</div>
      <div id="qdAdvisor" class="advisor-panel"><div class="ci">这里按这家店的历史曲线给出到店建议和时间里程碑。</div></div>
      <div id="qdEvidence" class="qbox mt16">
        <div class="cd-t"><b>📊 整合走势大图</b></div>
        <div id="qdPressChart" class="dash-chart mt16"><div class="empty">选门店后，这里把今天的叫号进度、排队压力和你的当天排队号画在同一张图上，悬停每个点可看全部数据。</div></div>
        <div id="qdDataSource" class="data-source mt16"></div>
      </div>
      <div id="qdInsights" class="qbox mt16">
        <div class="cd-t"><b>🗓 这家店的历史规律</b></div>
        <div id="qdCalledCurve" class="mt16"></div>
        <div id="qdHeatmap" class="mt16"></div>
        <div id="qdWeekday" class="mt16"></div>
        <div id="qdTrend" class="mt16"></div>
      </div>
      <div class="cd-t" style="margin:16px 0 8px">提醒 · 时间换算 · 采集配置</div>
      <div id="qdReminderCard" class="curve-sampling">
        <div>
          <b>🔔 提醒 <span class="tag read">只提醒 · 不取号</span></b>
          <p>两种提醒：已经有当天排队号时分几次提醒你出发；或每天按想吃的时间提醒你该取号了。都不会替你操作。</p>
          <div class="fl g8 fw mt8"><button class="chip on" id="remTabOnce" onclick="remTab('once')">当次 · 快叫到我时</button><button class="chip" id="remTabDaily" onclick="remTab('daily')">每日 · 该取号了</button></div>
          <div id="remOnce">
          <div class="fr mt8">
            <div class="fg"><label>提醒节奏</label><select id="qdrTemplate" onchange="renderReminderTemplateHint()"><option value="normal">标准 · 提前 80/50/25 号各提醒一次</option><option value="conservative">从容 · 提前 120/90/60/30 号</option><option value="urgent">临近 · 提前 50/25/10 号</option><option value="custom">自定义（在下方高级里填）</option></select></div>
            <div class="fg"><label>路上要多久（分钟，可选）</label><input id="qdrTravel" type="number" min="0" placeholder="如 25，用于推算出发时间"></div>
            <div class="fg"><label>备注（可选）</label><input id="qdrLabel" placeholder="如：我的当天排队号 / 帮朋友盯"></div>
          </div>
          <details class="adv mt8 advanced-only">
            <summary>高级 · 自定义提醒号码</summary>
            <div class="fg mt8"><label>叫到哪些号时提醒（逗号分隔，需小于你的当天排队号）</label><input id="qdrPoints" placeholder="如 1000,1025,1050" oninput="renderReminderTemplateHint()"></div>
            <p class="ps mt8">填了这里就按这些号码提醒，忽略上面的节奏。</p>
          </details>
          <div id="qdReminderStatus" class="mt8"><span class="mu">选门店并输入号码后，点「生成提醒」。</span></div>
          </div>
          <div id="remDaily" class="hid">
          <p class="mu mt8">按想吃饭的时间倒推取号窗口，每天提前提醒你手动取号；样本不足时不会乱提醒。<span class="tag auth">需要通知</span></p>
          <div class="fr mt8">
            <div class="fg"><label>门店</label><select id="nrStore"></select></div>
            <div class="fg"><label>想几点吃</label><input type="time" id="nrMeal" value="13:00"></div>
            <div class="fg"><label>路上要多久（分钟）</label><input type="number" id="nrTravel" value="0" min="0"></div>
            <div class="fg"><label>提前几分钟提醒</label><input type="number" id="nrSafety" value="10" min="0"></div>
            <div class="fg" style="align-self:flex-end"><button class="bt bt-r bt-s" onclick="saveNetTicketRoutine(true)">启用每日提醒</button></div>
            <div class="fg" style="align-self:flex-end"><button class="bt bt-o bt-s" onclick="saveNetTicketRoutine(false)">关闭</button></div>
          </div>
          <div id="nrStatus" class="pick-out mt8"><span class="mu">状态加载中…</span></div>
          </div>
        </div>
        <div class="curve-sampling-actions"><button class="bt bt-r bt-s" onclick="createTicketReminder()">🔔 生成提醒</button><button class="bt bt-w bt-s" id="qdrNotifyBtn" onclick="focusNotifySettings()">设置通知</button></div>
      </div>
      <details id="qdSamplingFold" class="card adv mt16 advanced-only">
        <summary><span class="setting-fold-title"><b>📡 本机持续采集</b><span>让「几点叫到」的判断越用越准；数据只留在本机，不上传。</span></span></summary>
        <div id="qdSamplingCard" class="curve-sampling mt16"><div><p style="margin-top:0">常用门店的公开排队曲线（叫号、等位）已默认自动记录，不需要通行证，越用越准；拿通行证后还能额外采集可约时段。</p></div><div class="curve-sampling-actions"><button class="bt bt-w bt-s" onclick="openSettingsFold('fold-sm')">详细配置</button></div></div>
      </details>
    </div>
  </section>
  <section id="p-qt" class="hid">
    <div class="cd">
      <div class="page-lead"><div><h2 class="ph">现在去吃 <span class="pm" data-kind="salmon" data-size="32"></span></h2><p class="ph-sub"><span class="tag read">只读 · 直接用</span> 这里看今天的排队：门店是否营业、前面还有几桌、大概等多久。简化版不会替你取号。</p></div><div class="fl g8 fw"><button class="bt bt-w bt-s" onclick="refreshQueueView()">刷新</button></div></div>
      <div class="fg"><label>关注门店</label>
        <div class="fl g8 fw" style="align-items:center"><button class="bt bt-w bt-s" onclick="openStorePicker({selected:qtSelected,onConfirm:applyQueueStores})">+ 选择门店（全国）</button><span class="mu">从全国门店里搜城市/门店名直接勾选。</span></div>
        <div id="qtStores" class="chips mt8"><span class="mu">尚未选择门店</span></div>
      </div>
      <div id="qtRecommend" class="mb16"></div>
      <div id="qtLive" class="mt16"><div class="skeleton skk"></div></div>
      <div class="card mt16" id="qtNextSteps" style="border:1px dashed var(--line)">
        <div class="fl ai jb fw g8">
          <div><b>看完排队，下一步？</b><div class="mu mt4">选好想去的店后，这两个常用动作能帮你定行程。</div></div>
        </div>
        <div class="fl g8 fw mt12">
          <button class="bt bt-w bt-s" onclick="go('qd')">⏱ 算几点叫到我（已拿号）</button>
          <button class="bt bt-w bt-s" onclick="enterAdvanced('ca')">📅 查未来可约时段（约未来）</button>
        </div>
      </div>
      <details class="adv mt16 advanced-only" open>
      <summary>自动取号计划（会执行操作）</summary>
      <div class="qbox mt16">
        <div class="fl ai jb fw g8"><label style="margin:0">取号计划 <span class="tag action">会执行操作</span></label><span class="mu">定时到点或一开放就自动远程取号，启用前会再次确认。</span></div>
        <div class="fl g8 fw mt8">
          <div class="fg"><label>门店</label><select id="ntStore"></select></div>
          <div class="fg"><label>触发方式</label><select id="ntMode" onchange="onNtModeChange()"><option value="time">到点取号</option><option value="on_open">一开放就取号</option></select></div>
          <div class="fg" id="ntTimeWrap"><label>几点取号</label><input type="time" id="ntTime"></div>
          <div class="fg" style="align-self:flex-end"><button class="bt bt-o bt-s" onclick="saveNetTicketPlan(true)">启用</button></div>
          <div class="fg" style="align-self:flex-end"><button class="bt bt-o bt-s" onclick="saveNetTicketPlan(false)">取消计划</button></div>
          <div class="fg" style="align-self:flex-end"><button class="bt bt-w bt-s" onclick="recoverNetTicketStatus()">恢复当前排队号</button></div>
        </div>
        <div id="ntStatus" class="pick-out mt8"><span class="mu">选门店和触发方式后，可启用自动取号计划；取到的号会显示在「我的单据」。启用前会再次确认。</span></div>
        <div class="fl ai g8 fw mt16" style="border-top:1px dashed var(--line);padding-top:12px"><span class="tag action">危险</span><button class="bt bt-o bt-s" onclick="cancelNetTicket()">取消排队号…</button><span class="mu">取消的是寿司郎小程序里的真实排队号，不可恢复；只想停止本工具请用「取消计划」。</span></div>
      </div>
      </details>
    </div>
  </section>

  <section id="p-sn" class="hid advanced-page">
    <div class="cd">
      <div class="page-lead"><div><h2 class="ph">自动抢预约 <span class="pm" data-kind="ebi" data-size="32"></span></h2><p class="ph-sub"><span class="tag action">会执行操作</span> 这里抢的是未来预约，不是当天排队号。已放出的可约时段可以立即抢；还没放出的时段可以「蹲未来预约时段」。启动前都会确认，抢到即停。</p></div></div>
      <div class="qbox mb16">
        <div class="fl ai jb fw g8"><label style="margin:0">自动抢预约（已放出的时段）</label><span class="mu">按你的门店和时段偏好扫描可约日历，抢到第一个符合的未来预约就停止。</span></div>
        <div class="fl g8 fw mt8"><button class="bt bt-r bt-s" onclick="sB()">按偏好抢预约</button><button class="bt bt-w bt-s" onclick="go('ca')">先看可约日历</button><button class="bt bt-w bt-s" onclick="expandSnPrefs()">改预约偏好</button></div>
      </div>
      <div class="qbox mb16">
        <div class="fl ai jb fw g8"><label style="margin:0">蹲未来预约时段（还没放出）</label><div class="fl g8 fw"><button class="bt bt-w bt-s" onclick="addSn()">添加目标时段</button><button class="bt bt-r bt-s" onclick="saveSn()">保存计划</button><button class="bt bt-y bt-s" onclick="startSn()">启动蹲未来预约</button></div></div>
        <div class="mu mt8">指定日期、门店、时间窗，开放瞬间自动尝试创建未来预约；抢到第一个就停。</div>
        <div id="snRows" class="mt8"></div>
        <div id="snPlan" class="mt16"><div class="empty">还没有蹲未来预约目标。点“添加目标时段”，填日期、门店和时间窗。</div></div>
      </div>
      <details class="adv mb16" id="snPrefs">
        <summary>预约/取号偏好 · 人数与门店优先级（自动抢预约和远程取号都用它）</summary>
        <div class="mt16">
        <div class="fr mb16">
          <div class="fg"><label>成人</label><input type="number" id="pa" min="0" max="10" value="2"></div>
          <div class="fg"><label>儿童</label><input type="number" id="pc" min="0" max="10" value="0"></div>
          <div class="fg"><label>桌型</label><select id="pt"><option value="T">桌位</option><option value="C">吧台</option></select></div>
          <div class="fg"><label>预约用手机号（可选）</label><input type="tel" id="pphone" maxlength="11" placeholder="11 位完整号码；留空用通行证里的号码"></div>
        </div>
        <div class="fg"><label>添加门店（搜全国）</label><div class="fl g8 fw"><input id="storeSearch" placeholder="输入城市或门店名，如 北京 / 凯德" style="flex:1;min-width:200px" onkeydown="if(event.key==='Enter'){searchStores();return false}"><button class="bt bt-w bt-s" onclick="searchStores()">搜索</button></div><div id="storeSearchResults" class="mt8"></div></div>
        <div class="fg"><label>预约/取号门店与优先级</label><div id="bookingStores" class="store-list"><span class="mu">用上方搜索添加，或拿到通行证后自动带入</span></div><div class="ps mt8">自动抢预约 / 远程取号会按勾选门店的排序依次尝试。新加的门店若从没在小程序点过，建议刷新凭证后先试一家确认可用。</div></div>
        <div class="fl ai g8 fw mt8"><button class="bt bt-r" onclick="sP()">保存全部偏好</button><span class="mu">人数、门店、时段是同一份偏好；在任意一个折叠里点保存，两边都会一起保存。</span></div>
        </div>
      </details>
      <details class="adv mb16" id="snPrefsTime">
        <summary>预约偏好 · 日期与时段策略</summary>
        <div class="mt16">
        <div class="preset-grid">
          <button class="preset" onclick="applyPreset('weekday_dinner')">工作日晚餐</button>
          <button class="preset" onclick="applyPreset('weekend_lunch')">周末午餐</button>
          <button class="preset" onclick="applyPreset('weekend_dinner')">周末晚餐</button>
          <button class="preset" onclick="applyPreset('any_available')">有时段就要</button>
        </div>
        <div class="fr mb16">
          <div class="fg"><label>日期优先级</label><select id="ppm"><option value="date">按日期优先</option><option value="weekend_first">周末优先</option><option value="weekday_first">工作日优先</option></select></div>
          <div class="fg"><label>时段策略</label><select id="pst"><option value="earliest">最早可约</option><option value="latest">最晚可约</option><option value="closest">接近目标时间</option></select></div>
          <div class="fg"><label>目标时间（如 1930 = 19:30）</label><input type="text" id="ptm" placeholder="1930"></div>
        </div>
        <div class="fg"><label>工作日时段</label><div id="wd" class="tl"></div><span class="at" onclick="aT('wd')">添加时段</span></div>
        <div class="fg"><label>周六时段</label><div id="sa" class="tl"></div><span class="at" onclick="aT('sa')">添加时段</span></div>
        <div class="fg"><label>周日时段</label><div id="su" class="tl"></div><span class="at" onclick="aT('su')">添加时段</span></div>
        <div class="fl ai g8 fw mt8"><button class="bt bt-r" onclick="sP()">保存全部偏好</button><span class="mu">人数、门店、时段是同一份偏好；在任意一个折叠里点保存，两边都会一起保存。</span></div>
        </div>
      </details>
    </div>
  </section>

  <section id="p-re" class="hid advanced-page">
    <div class="cd"><h2 class="ph">我的未来预约 / 当天排队号 <span class="pm" data-kind="maki" data-size="32"></span></h2><p class="ph-sub mb16"><span class="tag auth">需要通行证 🎫</span> 未来预约按预约时间到店；当天排队号按店内叫号进度到店。取消按钮是危险操作，会单独确认。</p><div id="rc"><div class="empty">加载中</div></div></div>
  </section>

  <section id="p-se" class="hid">
    <div class="settings-grid">
      <div class="cd settings-wide">
        <div class="page-lead"><div><h2 class="ph">设置 <span class="pm" data-kind="tamago" data-size="32"></span></h2><p class="ph-sub">按从上到下分区配置：凭证 → 通知 → 数据 → 排障。顶部状态条红色要处理，黄色按需配置。</p></div></div>
        <div id="settingsStatus"><div class="ci">状态加载中</div></div>
        <div class="mode-settings mt16" id="uiModeSettings">
          <div><b>界面模式</b><p class="mu mt8">简化版只保留必要看板；进阶版显示完整预约、取号、采集和维护功能。</p></div>
          <div class="mode-switch" aria-label="设置界面模式">
            <button type="button" onclick="setUIMode('simple')">简化版</button>
            <button type="button" onclick="setUIMode('advanced')">进阶版</button>
          </div>
        </div>
      </div>
      <div class="sect-divider"><span class="sect-no">1</span><b>凭证与认证</b><span class="mu">寿司郎通行证抢号取号才需要；GitHub 登录只用于补强线上排队基准。</span></div>
      <details class="cd setting-fold settings-wide" open>
        <summary><span class="setting-fold-title"><b>寿司郎通行证（认证凭证）</b><span>通行证不是排队号；只在抢未来预约、远程取号、读取我的单据时需要。它会被寿司郎定期回收，也可能被手机端重新打开小程序后顶掉。</span></span></summary>
        <div class="setting-fold-body">
        <div class="fl ai jb mb16 fw g8"><div class="cd-t" style="margin-bottom:0">通行证状态</div><div class="fl g8 fw"><button class="bt bt-r bt-s" onclick="openAuthWizard()">拿通行证（向导）</button><button class="bt bt-o bt-s" onclick="resetAuthOnly(true)">重置认证</button><button class="bt bt-w bt-s" onclick="testAuthProbe()">测试基础接口</button></div></div>
        <div class="ps">遇到 E010/error.server、401/403、远程取号失败或我的单据读不到时，优先点“重置认证”，再重新获取凭证。重置只清理本机保存的凭证，不会取消你已经拿到的预约或排队号。</div>
        <div id="mobileAuthState" class="diag-detail mt8">尚未加载</div>
        </div>
      </details>
      <details class="cd setting-fold settings-wide advanced-only">
        <summary><span class="setting-fold-title"><b>GitHub 登录与线上基准</b><span>GitHub 只用于认证线上排队基准服务，和寿司郎小程序凭证不是一回事。</span></span></summary>
        <div class="setting-fold-body">
        <div id="cloudState" class="diag-detail">尚未加载</div>
        <div class="fl g8 fw mt16"><button class="bt bt-r bt-s" onclick="startCloudLogin()">用 GitHub 登录</button><button class="bt bt-w bt-s" onclick="testCloudAuth()">验证连接</button><button class="bt bt-o bt-s" onclick="logoutCloudAuth()">退出云端</button></div>
        <div class="ps mt8">登录后可以读取线上基准来补强排队压力和到店预测；不会取号、取消号，也不会把数据库凭证写入本机。</div>
        <details class="btn-more mt16"><summary></summary><div class="fg mt8"><label>云端服务地址</label><input type="url" id="cloudUrl" placeholder="https://sushiro-cloud.your-name.workers.dev"></div><div class="fl g8 fw mt8"><button class="bt bt-r bt-s" onclick="saveCloudAuth()">保存服务地址</button></div><div class="ps mt8">仅自建或排障时需要。线上数据库凭证只应保存在云端服务 secrets 里。</div></details>
        </div>
      </details>
      <div class="sect-divider"><span class="sect-no">2</span><b>通知</b><span class="mu">抢到预约、叫号提醒会用这里推送；至少配一个渠道。</span></div>
      <details class="cd setting-fold" open>
        <summary><span class="setting-fold-title"><b>通知渠道</b><span>配置飞书、Telegram、Bark 或 Server酱；抢到预约、叫号提醒会用这里推送。</span></span></summary>
        <div class="setting-fold-body">
        <div class="fg"><label>飞书 Webhook</label><input type="text" id="nf" placeholder="https://open.feishu.cn/..."></div>
        <div class="fr"><div class="fg" style="flex:1"><label>Telegram Token</label><input type="text" id="ntt" placeholder="123456:ABC..."></div><div class="fg" style="flex:1"><label>Chat ID</label><input type="text" id="ntc" placeholder="-100..."></div></div>
        <div class="fr"><div class="fg" style="flex:1"><label>Bark URL</label><input type="text" id="nbu" placeholder="https://api.day.app"></div><div class="fg" style="flex:1"><label>Bark Key</label><input type="text" id="nbk"></div></div>
        <div class="fg"><label>Server 酱 Key</label><input type="text" id="ns" placeholder="SCT..."></div>
        <div class="fl g8 fw mt8"><button class="bt bt-r" onclick="sN()">保存通知</button><button class="bt bt-w" onclick="tN('all')">保存并测试全部</button></div>
        <details class="btn-more mt8"><summary></summary><div class="ps mt8">单渠道测试会先保存当前表单再发送。</div><div class="fl g8 fw mt8"><button class="bt bt-w bt-s" onclick="tN('feishu')">测试飞书</button><button class="bt bt-w bt-s" onclick="tN('telegram')">测试 Telegram</button><button class="bt bt-w bt-s" onclick="tN('bark')">测试 Bark</button><button class="bt bt-w bt-s" onclick="tN('serverchan')">测试 Server酱</button></div></details>
        </div>
      </details>
      <div class="sect-divider advanced-only"><span class="sect-no">3</span><b>数据与预测</b><span class="mu">让「几点叫到」更准；看历史规律反推更值得抢的时段。</span></div>
      <details class="cd setting-fold settings-wide advanced-only" id="fold-sm" ontoggle="if(this.open)lSm()">
        <summary><span class="setting-fold-title"><b>预测准确度 <span class="pm" data-kind="unagi" data-size="26"></span></b><span>提升“几点叫到、几点出发”的判断；常用门店的公开排队曲线已默认自动记录，这里只在想更准时配置。</span></span></summary>
        <div class="setting-fold-body">
        <div class="fl g8 fw"><button class="bt bt-w bt-s" onclick="runSampleOnce()">收集一次</button><button class="bt bt-r bt-s" onclick="startSampling()">开启持续采集</button></div>
        <div class="sample-grid mt16">
          <label class="check"><input type="checkbox" id="spEnabled">开启本机持续采集</label>
          <label class="check debug-only"><input type="checkbox" id="spAuto">应用启动后自动收集</label>
          <div class="fg debug-only"><label>间隔秒数</label><input type="number" id="spInterval" min="60" step="60" value="300"></div>
          <div class="fg debug-only"><label>开始</label><input type="time" id="spStart" value="10:00"></div>
          <div class="fg debug-only"><label>结束</label><input type="time" id="spEnd" value="22:00"></div>
        </div>
        <div class="fg"><label>提升哪家店的预测</label><div id="samplingStores" class="chips"><span class="mu">加载中</span></div><div id="sampleStoreHint" class="ps mt8"></div></div>
        <div class="fl g8 fw"><button class="bt bt-r" onclick="saveSampling()">保存预测配置</button><button class="bt bt-w" onclick="usePrefSamplingStores()">跟随预约偏好门店</button></div>
        <div id="sampleState" class="sample-state"><div class="ci">尚未加载</div></div>
        <div id="sampleResult" class="diag-detail hid"></div>
        </div>
      </details>
      <details class="cd setting-fold settings-wide advanced-only" id="fold-in" ontoggle="if(this.open)lI()">
        <summary><span class="setting-fold-title"><b>历史洞察 <span class="pm" data-kind="kappa" data-size="26"></span></b><span>按门店、星期、时段统计开放概率和售罄速度，反推更值得抢的目标。</span></span></summary>
        <div class="setting-fold-body">
        <div class="fl g8 fw mb16"><button class="bt bt-w bt-s" onclick="lI()">刷新</button></div>
        <div id="ic"><div class="empty">加载中</div></div>
        </div>
      </details>
      <div class="sect-divider advanced-only"><span class="sect-no">4</span><b>排障</b><span class="mu">状态异常、代理残留、看日志；平时不用动。</span></div>
      <details class="cd setting-fold settings-wide advanced-only" id="fold-safe">
        <summary><span class="setting-fold-title"><b>诊断与维护</b><span>状态异常、代理残留、需要复制诊断时打开。</span></span></summary>
        <div class="setting-fold-body">
        <div class="fl ai jb mb16 fw g8"><div class="cd-t" style="margin-bottom:0">本机诊断</div><div class="fl g8 fw"><button class="bt bt-w bt-s" onclick="lD()">刷新</button><button class="bt bt-w bt-s" onclick="copyDiag()">复制诊断</button><button class="bt bt-r bt-s" onclick="repairP()">修复代理</button><button class="bt bt-w bt-s" onclick="rST()">重置抓包</button></div></div>
        <div id="diagNext" class="diag-next warn"><h3>先处理这件事</h3><p>刷新诊断后会显示最值得先做的一步。</p></div>
        <div id="dg" class="cg"><div class="ci">尚未加载</div></div>
        <div id="ddetail" class="diag-detail hid"></div>
        </div>
      </details>
      <details class="cd setting-fold settings-wide advanced-only" id="fold-lo" ontoggle="if(this.open)lL()">
        <summary><span class="setting-fold-title"><b>运行日志 <span class="pm" data-kind="maguro" data-mood="sleep" data-size="26"></span></b><span>排障时看；平时不用展开。</span></span></summary>
        <div class="setting-fold-body"><div class="lg" id="lv"></div></div>
      </details>
      <div class="danger-zone settings-wide advanced-only">
        <div class="danger-zone-head"><b>⚠ 危险操作</b><span class="mu">不可恢复，执行前会再次确认。与上方日常配置隔离。</span></div>
        <div class="fl g8 fw mt8"><button class="bt bt-o bt-s" onclick="stopProcesses()">停止本应用进程</button><button class="bt bt-o bt-s" onclick="uninstallAll()">卸载清理</button></div>
      </div>
    </div>
  </section>

</main>
<div class="belt" id="belt" aria-hidden="true"></div>
<footer class="ft">由 <a href="https://github.com/Ryujoxys/sushiro-overdose">sushiro-overdose</a> 驱动 · 非官方工具，仅供学习</footer>

<script>
let cp='da',es={status:'idle'},hc=0,as=[],sd='',pr={},pf='',cE=null,stores=[],selStores=[],calErrs=[],arTimer=null,lastDiag=null,spCfg={},spState={status:'idle'},spAutoStart={},spQueueState={},qdSelected=[],qdDashboardData={},qtSelected=[],qtTrendStores=[],qaStatus={},ah={},nfc=true,notifyChannels=[],cloudAuth={},cloudVerifyOnLoad=false,cloudRefreshPending=false,_inflight=null,qdAutoTimer=null,qdRefreshToken=0,qdDashToken=0,uiMode='simple',uiModeSeq=0,prefsLoaded=false,prefsLoading=null;
const W=['日','一','二','三','四','五','六'];
const need=['x_app_code','query_auth','reservation_auth','user_agent','referer','wechat_id','phone_number','store_ids'];
const csrfToken=document.querySelector('meta[name="sushiro-csrf"]')?.content||'';
const rawFetch=window.fetch.bind(window);
function sameOriginRequest(input){
  try{
    const target=input instanceof Request?input.url:String(input);
    return new URL(target,location.href).origin===location.origin;
  }catch(e){return true}
}
let staleSessionReloading=false;
window.fetch=async(input,init)=>{
  const opt=init?{...init}:{};
  const method=String(opt.method||(input&&input.method)||'GET').toUpperCase();
  if((method==='POST'||method==='PUT')&&sameOriginRequest(input)){
    const h=new Headers(opt.headers||(input&&input.headers)||{});
    h.set('X-Sushiro-CSRF',csrfToken);
    opt.headers=h;
  }
  const resp=await rawFetch(input,opt);
  // 应用重启后会换 CSRF token：旧页面提交会 403。自动刷新拿新页面，避免用户卡在“CSRF 校验失败”。
  if(resp.status===403&&!staleSessionReloading&&sameOriginRequest(input)){
    try{const d=await resp.clone().json();if(/CSRF/i.test(String(d&&d.error||''))){staleSessionReloading=true;toast('应用已重启，页面已过期，正在自动刷新…');setTimeout(()=>location.reload(),1200)}}catch(e){}
  }
  return resp;
};
function el(id){return document.getElementById(id)}
function esc(s){const d=document.createElement('div');d.textContent=s==null?'':String(s);return d.innerHTML}
function toast(msg,type){if(msg==null||msg==='')return;const s=String(msg);if(!type)type=/失败|错误|不可|无法|未能|超时|缺|invalid|error/i.test(s)?'err':(/请先|请填|请至少|至少填|请选|尚未/.test(s)?'warn':(/已|成功|完成|保存|启用|清理|恢复|启动/.test(s)?'ok':'info'));let w=el('toastWrap');if(!w){w=document.createElement('div');w.id='toastWrap';w.className='toast-wrap';document.body.appendChild(w)}const t=document.createElement('div');t.className='toast '+type;t.textContent=s;w.appendChild(t);requestAnimationFrame(()=>t.classList.add('in'));const long=/失败|错误|不可|无法|未能|超时|invalid|error/i.test(s);const ms=long?6500:2900;let timer=setTimeout(()=>{t.classList.remove('in');setTimeout(()=>t.remove(),280)},ms);t.onclick=()=>{clearTimeout(timer);t.classList.remove('in');setTimeout(()=>t.remove(),280)};t.title='点此关闭'}
function submitting(key){return _inflight&&_inflight.has(key)}
let _qdInputTimer=null;
function qdInputDebounced(){clearTimeout(_qdInputTimer);_qdInputTimer=setTimeout(()=>{renderReminderTemplateHint();loadQueueDashboard()},400)}
async function submitGuard(key,fn){if(!_inflight)_inflight=new Set();if(_inflight.has(key)){toast('正在处理，请稍候…','warn');return}const btn=document.activeElement;if(btn&&btn.tagName==='BUTTON'){btn.dataset._oldTxt=btn.textContent;btn.disabled=true;btn.textContent='提交中…'}_inflight.add(key);try{await fn()}finally{_inflight.delete(key);if(btn&&btn.tagName==='BUTTON'&&btn.dataset._oldTxt!=null){btn.disabled=false;btn.textContent=btn.dataset._oldTxt;delete btn.dataset._oldTxt}}}
function confirmDialog(opts){opts=typeof opts==='string'?{body:opts}:(opts||{});const danger=opts.danger!=null?opts.danger:/危险|不可恢复|卸载|清理本地|删除/.test(opts.body||'');return new Promise(res=>{let ov=el('confirmOv');if(!ov){ov=document.createElement('div');ov.id='confirmOv';ov.className='ov';document.body.appendChild(ov)}ov.innerHTML='<div class="ovc confirm-ovc'+(danger?' confirm-danger':'')+'"><div class="confirm-h">'+(danger?'⚠ ':'')+esc(opts.title||(danger?'危险操作':'请确认'))+'</div><div class="confirm-b">'+esc(opts.body||'')+'</div><div class="confirm-acts"><button class="bt bt-w" id="cfNo">'+esc(opts.cancel||'取消')+'</button><button class="bt bt-r" id="cfYes">'+esc(opts.ok||(danger?'确认':'继续'))+'</button></div></div>';ov.classList.remove('hid');ov.style.display='flex';const done=v=>{ov.classList.add('hid');ov.style.display='none';res(v)};el('cfYes').onclick=()=>done(true);el('cfNo').onclick=()=>done(false);el('cfYes').focus();ov.onclick=e=>{if(e.target===ov)done(false)}})}
// ensureNotifyConfigured 在写操作（抢预约/蹲未来/生成提醒）前校验通知渠道是否配置。
// 已配置→返回 true 继续；未配置→弹 confirmDialog 引导去配置：「去配置」返回 false（中断当前操作、跳设置页），
// 「先继续」返回 true（操作照常，但用户已被提醒收不到推送）。actionLabel 用于文案，如"抢到预约"。
async function ensureNotifyConfigured(actionLabel){if(nfc)return true;const go=await confirmDialog({title:'还没配置通知渠道',body:'通知渠道（飞书/Telegram/Bark/Server酱）没配的话，'+(actionLabel||('操作成功')+'后')+'你收不到推送，得一直盯着屏幕。现在去配一个？只需填一次。',ok:'去配置通知',cancel:'先继续'});if(go){focusNotifySettings();return false}return true}
const OV_CLOSERS={confirmOv:()=>{const n=el('cfNo');if(n)n.click()},storePicker:()=>closeStorePicker(),healthPanel:()=>closeHealthPanel(),firstUse:()=>closeFirstUseWizard(),authWiz:()=>closeAuthWizard()};
document.addEventListener('keydown',e=>{if(e.key!=='Escape')return;const open=Array.from(document.querySelectorAll('.ov')).filter(x=>!x.classList.contains('hid')&&x.style.display!=='none'),ov=open[open.length-1];if(ov&&OV_CLOSERS[ov.id]){e.preventDefault();OV_CLOSERS[ov.id]()}});
async function safeFetch(url,opts,timeoutMs){
  const ms=typeof timeoutMs==='number'?timeoutMs:15000;
  const ctrl=new AbortController();
  const t=setTimeout(()=>ctrl.abort(),ms);
  try{
    const r=await fetch(url,{...(opts||{}),signal:ctrl.signal});
    if(!r.ok){let body='';try{body=(await r.text()).slice(0,500)}catch(e){}
      throw new Error('HTTP '+r.status+' '+r.statusText+(body?' — '+body:''));
    }
    return await r.json();
  }catch(e){
    if(e.name==='AbortError')throw new Error('请求超时（'+ms+'ms）: '+url);
    throw e;
  }finally{clearTimeout(t)}
}
function loadErrBoxHTML(err,retryAttr,label){
  const msg=String((err&&(err.message||err))||'(unknown)');
  const head=label?label+'失败':'加载失败';
  return '<div class="empty"><b>'+esc(head)+'</b><br><code style="word-break:break-all;display:inline-block;margin-top:6px;color:var(--red)">'+esc(msg)+'</code>'+(retryAttr?'<div class="mt8"><button class="bt bt-w bt-s" onclick="'+retryAttr+'">重试</button></div>':'')+'</div>';
}
function escA(s){return esc(s).replaceAll('"','&quot;')}
const NAV_GROUPS=[
  {id:'home',label:'首页',pages:[['da','概览']]},
  {id:'eat',label:'现在去吃',pages:[['qt','门店排队']]},
  {id:'number',label:'我有号码',pages:[['qd','叫号预测']]},
  {id:'book',label:'约未来',pages:[['ca','可约日历'],['sn','自动抢预约']]},
  {id:'mine',label:'我的单据',pages:[['re','预约 / 排队号']]},
  {id:'settings',label:'设置',pages:[['se','设置']]}
];
const PAGE_GROUP={};NAV_GROUPS.forEach(g=>g.pages.forEach(([p])=>PAGE_GROUP[p]=g.id));
const ADVANCED_GROUPS=new Set(['book','mine']);
const ADVANCED_PAGES=new Set(['ca','sn','re']);
const ADVANCED_FOLDS=new Set(['fold-sm','fold-in','fold-lo','fold-safe']);
function currentUIMode(){return uiMode==='advanced'?'advanced':'simple'}
function isAdvancedPage(page){return ADVANCED_PAGES.has(page)}
function modeLabel(){return currentUIMode()==='advanced'?'进阶版':'简化版'}
function cachedUIMode(){try{return localStorage.getItem('sushiro_ui_mode')==='advanced'?'advanced':'simple'}catch(e){return 'simple'}}
function cacheUIMode(mode){uiMode=mode==='advanced'?'advanced':'simple';try{localStorage.setItem('sushiro_ui_mode',uiMode)}catch(e){}}
function applyUIMode(){
 uiMode=currentUIMode();
 document.body.classList.toggle('advanced-mode',uiMode==='advanced');
 document.body.classList.toggle('simple-mode',uiMode!=='advanced');
 document.querySelectorAll('#uiModeSwitch button,.mode-settings .mode-switch button').forEach(b=>{
  const adv=/进阶|advanced/i.test(b.id||b.textContent||'');
  b.classList.toggle('on',adv?uiMode==='advanced':uiMode!=='advanced');
 });
 document.querySelectorAll('.nav.top a').forEach(a=>{
  const hidden=uiMode!=='advanced'&&ADVANCED_GROUPS.has(a.dataset.group||'');
  a.classList.toggle('hid',hidden);
 });
 renderSettingsStatus();
 if(uiMode!=='advanced'&&isAdvancedPage(cp))go('da');
}
async function persistUIMode(mode){
 const wanted=mode==='advanced'?'advanced':'simple';
 try{
  const base=await ensurePrefsLoaded();
  if(currentUIMode()!==wanted)return;
  const d=await(await fetch('/api/preferences',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({...base,ui_mode:wanted})})).json();
  if(d.error){toast('模式已切换，但保存失败：'+d.error);return}
  if(currentUIMode()!==wanted)return;
  pr=d.preferences||{...base,ui_mode:wanted};prefsLoaded=true;cacheUIMode(pr.ui_mode==='advanced'?'advanced':'simple');applyUIMode();
 }catch(e){toast('模式已切换，但保存失败，下次打开可能恢复原设置')}
}
async function setUIMode(mode){
 uiModeSeq++;cacheUIMode(mode);applyUIMode();
 toast('已切换到'+modeLabel());
 persistUIMode(uiMode);
}
async function enterAdvanced(target){
 if(currentUIMode()==='advanced'){if(target)go(target);return}
 if(!await confirmDialog({title:'切换到进阶版？',body:'该功能在进阶版中。进阶版会显示完整预约、取号、采集和维护功能；会执行操作仍会单独确认。',ok:'切换到进阶版',cancel:'留在简化版'}))return;
 await setUIMode('advanced');
 if(target)go(target);
}
function renderSubnav(g,active){const sn=el('subnav');if(!sn)return;if(!g||g.pages.length<=1){sn.innerHTML='';sn.classList.add('hid');return}sn.classList.remove('hid');sn.innerHTML=g.pages.map(([p,label])=>'<a href="#" class="'+(p===active?'on':'')+'" onclick="go(\''+p+'\');return false">'+esc(label)+'</a>').join('')}
function goGroup(gid){const g=NAV_GROUPS.find(x=>x.id===gid);if(g)go(g.pages[0][0]);return false}
function stopQDAutoRefresh(){if(qdAutoTimer){clearInterval(qdAutoTimer);qdAutoTimer=null}}
function stopCalendarAutoRefresh(){if(arTimer){clearInterval(arTimer);arTimer=null}}
function go(n,e,noPush){if(!PAGE_GROUP[n])n='da';if(currentUIMode()!=='advanced'&&isAdvancedPage(n)){toast('该功能在进阶版中，可切换后使用','warn');if(!noPush&&location.hash.slice(1)!=='da')history.pushState(null,'','#da');n='da'}if(cp==='qd'&&n!=='qd')stopQDAutoRefresh();if(cp==='ca'&&n!=='ca')stopCalendarAutoRefresh();document.querySelectorAll('.wrap>section[id^="p-"]').forEach(p=>p.classList.add('hid'));const sec=el('p-'+n);if(sec)sec.classList.remove('hid');const gid=PAGE_GROUP[n]||'home',g=NAV_GROUPS.find(x=>x.id===gid);document.querySelectorAll('.nav.top a').forEach(a=>a.classList.toggle('on',a.dataset.group===gid));renderSubnav(g,n);cp=n;if(!noPush&&location.hash.slice(1)!==n)history.pushState(null,'','#'+n);const loader=({da:lDA,ca:lC,qd:lQD,qt:lQT,sn:lSn,re:lR,se:lS})[n];loader?.();if(cloudRefreshPending&&(n==='qd'||n==='qt')){cloudRefreshPending=false;setTimeout(refreshCloudDependentViews,120)}applyUIMode();return false}
window.addEventListener('popstate',()=>{const h=location.hash.slice(1);go(h&&PAGE_GROUP[h]?h:'da',null,true)});
async function loadStatus(){const v=el('ver');try{const r=await(await fetch('/api/status')).json();v.textContent='v'+r.version;v.classList.remove('hid');hc=!!r.has_config;pf=r.platform||'';es=r.engine||{status:'idle'};spState=r.sampling||spState;ah=r.auth_health||{};nfc=r.notify_configured!==false;maybeShowQuarantineCard(r);uE();uD();uAuth();renderSettingsStatus();loadActiveTickets(false);}catch(e){v.textContent='offline';v.classList.remove('hid');heroLoadFailed(e)}}
function heroLoadFailed(err){const badge=el('heroBadge'),t=el('heroTitle'),c=el('heroCopy');if(badge)badge.textContent='连接异常';if(t)t.textContent='读不到运行状态';if(c)c.innerHTML='本机服务没有响应：<code style="word-break:break-all">'+esc(String((err&&err.message)||err||'unknown'))+'</code> <button class="bt bt-w bt-s" onclick="loadStatus()">重试</button>'}
function uAuth(){
 const pill=el('authPill'),banner=el('authBanner'),st=(ah&&ah.status)||'unknown',reason=(ah&&ah.reason)?String(ah.reason):'';
 if(pill){let cls='authpill',txt='';
  if(!hc){txt='只读模式'}
  else if(st==='stale'){cls+=' stale';txt='通行证可能失效'}
  else if(!nfc){cls+=' warn';txt='通知未配置'}
  else{cls+=' ok';txt='一切就绪'}
  pill.className=cls;pill.textContent=txt;pill.classList.remove('hid')}
 if(banner){if(hc&&st==='stale'){banner.classList.remove('hid');banner.innerHTML='<b>🎫 通行证可能失效了</b>寿司郎会定期回收通行证，也可能被手机端重新登录顶掉。<button class="bt bt-r bt-s" onclick="event.stopPropagation();resetAuthAndStart()">重新获取（约 3 分钟）</button>'+(reason?'<details style="flex-basis:100%" onclick="event.stopPropagation()"><summary class="mu" style="cursor:pointer">技术细节</summary><code style="word-break:break-all">'+esc(reason)+'</code></details>':'')}else{banner.classList.add('hid');banner.innerHTML=''}}}
function healthStripHTML(items){return items.map(x=>'<div class="strip"><span class="st '+x.s+'">'+(x.s==='ok'?'✓':x.s==='bad'?'✕':'!')+'</span><div><b>'+esc(x.t)+'</b><span class="sd">'+esc(x.d)+'</span></div>'+(x.a?'<button class="bt bt-w bt-s" onclick="'+x.a.f+'">'+esc(x.a.l)+'</button>':'')+'</div>').join('')}
function openHealthPanel(){let ov=el('healthPanel');if(!ov){ov=document.createElement('div');ov.id='healthPanel';ov.className='ov';document.body.appendChild(ov)}
 const st=(ah&&ah.status)||'unknown',spOK=!!(spState.running||spState.enabled||spState.sample_runs>0);
 const items=[
  {t:'寿司郎通行证 🎫',d:hc?(st==='stale'?('可能已失效'+((ah&&ah.reason)?('：'+ah.reason):'，建议重新获取')):'已就绪'):'看排队不需要；抢未来预约、远程取号、读单据才需要',s:hc?(st==='stale'?'bad':'ok'):'warn',a:hc?{l:'重新获取',f:st==='stale'?'closeHealthPanel();resetAuthAndStart()':'closeHealthPanel();openAuthWizard()'}:{l:'去获取',f:'closeHealthPanel();openAuthWizard()'}},
  {t:'通知渠道',d:nfc?('已配置'+(notifyChannels.length?('：'+notifyChannels.join('、')):'')):'不配置就收不到叫号提醒和抢到通知',s:nfc?'ok':'warn',a:{l:nfc?'管理':'去配置',f:'closeHealthPanel();focusNotifySettings()'}}
 ];
 if(currentUIMode()==='advanced')items.push({t:'预测数据',d:spOK?'采集中，“几点叫到”会越来越准':'开启后到店预测更准（可选）',s:spOK?'ok':'warn',a:{l:spOK?'查看':'去开启',f:"closeHealthPanel();openSettingsFold('fold-sm')"}});
 ov.innerHTML='<div class="ovc" style="width:min(560px,96vw)"><div class="fl ai jb mb16"><b>运行前置条件</b><button class="bt bt-w bt-s" onclick="closeHealthPanel()">关闭</button></div>'+healthStripHTML(items)+'<p class="mu mt16">红色需要处理，黄色按需配置；任何页面点右上角胶囊都能回到这里。</p></div>';
 ov.onclick=e=>{if(e.target===ov)closeHealthPanel()};
 ov.classList.remove('hid');ov.style.display='flex'}
function closeHealthPanel(){const ov=el('healthPanel');if(ov){ov.classList.add('hid');ov.style.display='none'}}
function authPillClick(){openHealthPanel()}
async function init(){cacheUIMode(cachedUIMode());applyUIMode();consumeCloudAuthResult();fillPageMascots();buildBelt();await loadStatus();await lP();checkUpdate();sse();const h=location.hash.slice(1);if(h&&PAGE_GROUP[h]&&h!=='da')go(h);else{loadHomeLive(true);maybeShowIntro()}}
function consumeCloudAuthResult(){try{const p=new URLSearchParams(location.search);const connected=p.get('cloud_connected');if(connected){cloudRefreshPending=true;cloudVerifyOnLoad=true;toast('云端 GitHub 登录已完成');refreshCloudDependentViews()}if(p.get('cloud_error'))toast(p.get('cloud_error'));if(p.has('cloud_connected')||p.has('cloud_error'))history.replaceState(null,'',location.pathname+location.hash)}catch(e){}}
function refreshCloudDependentViews(){try{if(cp==='qd')loadQueueDashboard();if(cp==='qt')refreshQueueView()}catch(e){}}
function maybeShowIntro(){try{/* sushiro_intro_seen_v2：bump 版本键，重要改版后让所有用户（含已看过 v1 的）重新看到一次引导。
   老用户想随时重看可点 setupCard 的「打开新手引导」按钮。仍保留 hc 屏蔽：已有通行证的用户不被首启浮层打扰。 */if(hc)return;if(localStorage.getItem('sushiro_intro_seen_v2'))return;localStorage.setItem('sushiro_intro_seen_v2','1');openFirstUseWizard()}catch(e){}}
function maybeShowQuarantineCard(r){try{if(!r||!r.quarantined)return;if(localStorage.getItem('sushiro_q_dismissed'))return;const exe=r.executable_path||'';const cmd='xattr -dr com.apple.quarantine '+(exe||'/Applications/sushiro-overdose.app');let ov=el('quarantineOv');if(!ov){ov=document.createElement('div');ov.id='quarantineOv';ov.className='ov';document.body.appendChild(ov)}
ov.innerHTML='<div class="ovc" style="width:min(560px,96vw)"><div class="fl ai jb mb16"><b>⚠ macOS 隔离标记</b><button class="bt bt-w bt-s" onclick="dismissQuarantineCard()">关闭</button></div><p class="mu">这个 App 是从网上下载的，macOS 给它加了隔离标记（Gatekeeper）。一般不影响使用，但少数情况下会让系统代理设置、通知或抓包证书被拒。移除后更省心。</p><p class="mt8"><b>打开「终端」粘贴运行这一行（按回车）即可：</b></p><pre style="background:rgba(0,0,0,.06);padding:10px 12px;border-radius:8px;overflow:auto;font-size:13px;word-break:break-all;white-space:pre-wrap"><code id="quarantineCmd">'+esc(cmd)+'</code></pre><div class="fl g8 fw mt12"><button class="bt bt-s" onclick="copyQuarantineCmd()">复制命令</button><button class="bt bt-o bt-s" onclick="dismissQuarantineCard()">我已执行，不再提示</button></div><p class="mu mt12">运行后重启本工具即可。仅对本工具生效，不动其它 App。</p></div>';
ov.classList.remove('hid');ov.style.display='flex';ov.onclick=e=>{if(e.target===ov)dismissQuarantineCard()}}catch(e){}}
function dismissQuarantineCard(){const ov=el('quarantineOv');if(ov){ov.classList.add('hid');ov.style.display='none'}try{localStorage.setItem('sushiro_q_dismissed','1')}catch(e){}}
function copyQuarantineCmd(){const c=el('quarantineCmd');if(!c)return;const txt=c.textContent||'';if(navigator.clipboard&&navigator.clipboard.writeText){navigator.clipboard.writeText(txt).then(()=>toast('命令已复制，去终端粘贴运行'),()=>fallbackCopy(txt))}else{fallbackCopy(txt)}}
function fallbackCopy(txt){try{const ta=document.createElement('textarea');ta.value=txt;document.body.appendChild(ta);ta.select();document.execCommand('copy');document.body.removeChild(ta);toast('命令已复制，去终端粘贴运行')}catch(e){toast('复制失败，请手动选中上方命令复制')}}
function isRun(){return ['capturing','booking','sniping'].includes(es.status)}
function awzPeek(){try{const s=JSON.parse(localStorage.getItem('sushiro_wizard_state')||'null');if(!s)return null;const c=s.cap||{};return{step:s.step||1,fields:need.filter(k=>c[k]).length}}catch(e){return null}}
function renderSetupCard(){
 const card=el('setupCard'),list=el('setupList');if(!card||!list)return;
 const aw=awzPeek(),items=[];
 const authS=hc?((ah&&ah.status==='stale')?'warn':'ok'):'warn';
 items.push({t:'寿司郎通行证 🎫',d:hc?(authS==='warn'?'可能已失效，建议重新获取':'已就绪'):(aw&&aw.fields>0?('拿到一半（'+aw.fields+'/'+need.length+' 项），可以继续'):'抢未来预约、远程取号、读单据时才需要'),s:authS,a:hc?(authS==='warn'?{l:'重新获取',f:'resetAuthAndStart()'}:null):{l:(aw&&aw.fields>0)?'继续获取':'去获取',f:'openAuthWizard()'}});
 const hasStores=(pr.selected_stores||[]).length>0;
 items.push({t:'常用门店',d:hasStores?('已选 '+pr.selected_stores.length+' 家，各页面自动带入'):'选好后看排队、预测、日历都不用重选',s:hasStores?'ok':'warn',a:hasStores?null:{l:'去选店',f:'openGuestStorePicker()'}});
 items.push({t:'通知渠道',d:nfc?'已配置':'不配置就收不到叫号提醒和抢到通知',s:nfc?'ok':'warn',a:nfc?null:{l:'去配置',f:'focusNotifySettings()'}});
 const spOK=!!(spState.running||spState.enabled||spState.sample_runs>0);
 if(currentUIMode()==='advanced')items.push({t:'预测数据',d:spOK?'采集中，“几点叫到”会越来越准':'开启后到店预测更准（可选）',s:spOK?'ok':'warn',a:spOK?null:{l:'去开启',f:"openSettingsFold('fold-sm')"}});
 const allOK=items.every(x=>x.s==='ok');
 card.classList.toggle('hid',allOK);
 if(allOK)return;
 list.innerHTML=healthStripHTML(items);
}
function journeyStepHTML(kind,title,desc,state){
 const label={read:'只读',auth:'通行证',action:'会执行'}[kind]||kind;
 return'<div class="journey-step '+escA(kind)+' '+escA(state||'')+'"><span>'+esc(label)+'</span><b>'+esc(title)+'</b><small>'+esc(desc)+'</small></div>'
}
function journeyButtonsHTML(buttons){return(buttons||[]).map((b,i)=>'<button class="bt '+(i===0?'bt-r':'bt-w')+' bt-s" onclick="'+b.f+'">'+esc(b.l)+'</button>').join('')}
function renderJourneyPanel(){
 const box=el('journeyPanel');if(!box)return;
 const hasStores=(pr.selected_stores||[]).length>0,st=(ah&&ah.status)||'unknown',stale=hc&&st==='stale',running=isRun(),tickets=(activeTickets||[]).length;
 const aw=awzPeek(),authDesc=hc?(stale?'可能已失效，建议更新':'已就绪'):(aw&&aw.fields>0?('已拿到 '+aw.fields+'/'+need.length+' 项，可继续'):'抢预约前再拿');
 const actionDesc=running?'正在运行':(es.status==='error'?'需要先排障':(hc&&hasStores?'可以开始未来预约或远程取号':'先补齐门店和通行证'));
 let plan={level:'ok',mode:'只读优先',title:'今天该走哪条路',copy:'看排队、叫号预测可以直接用；抢未来预约、远程取号、读取我的单据才需要通行证；会提交到寿司郎的动作会再次确认。',buttons:[{l:'先看实时排队',f:"go('qt')"},{l:'拿通行证',f:'startAuth()'}]};
 if(tickets>0)plan={level:'ok',mode:'已有单据',title:'先看你手上的单据',copy:'你已经有未完成的未来预约或当天排队号。先确认叫号和记录，避免重复取号或重复预约。',buttons:[{l:'查看我的单据',f:"enterAdvanced('re')"},{l:'几点叫到我',f:"go('qd')"}]};
 else if(es.status==='error'){const certUAC=/机器级|LocalMachine|管理员权限|UAC|RunAs|elevated|exit code/i.test(es.message||'');plan={level:'bad',mode:'需要处理',title:'先处理这件事',copy:explainMsg(es.message||'')+' 处理前不会自动取消你的未来预约或当天排队号。',buttons:certUAC?[{l:'重新装证书（会弹UAC，点“是”）',f:'startAuth()'},{l:'改用手机抓包（更稳）',f:'closeAuthWizard();openAuthWizard();setTimeout(()=>awzDevice("ios"),50)'},{l:'打开本机诊断',f:'openDiagnostics()'}]:[{l:'打开本机诊断',f:'openDiagnostics()'},{l:hc?'重新拿通行证':'拿通行证',f:'startAuth()}'}]};}
 else if(running)plan={level:'warn',mode:'运行中',title:'当前有任务正在执行',copy:'页面可以保持打开；想看细节去运行日志，想换目标前先停止当前任务。',buttons:[{l:'查看运行日志',f:"openSettingsFold('fold-lo')"},{l:'停止当前任务',f:'sE()'}]};
 else if(!hc)plan={level:'warn',mode:'只读可用',title:'先不用登录，也能看排队',copy:'现在去吃、叫号预测都不需要通行证。等你要抢未来预约、远程取当天排队号或读取我的单据时，再按引导拿通行证。',buttons:[{l:'选门店看排队',f:'openGuestStorePicker()'},{l:'我要抢未来预约：拿通行证',f:'startAuth()'}]};
 else if(stale)plan={level:'bad',mode:'通行证待更新',title:'通行证可能失效了',copy:'看排队仍然能用；抢预约、远程取号、读取单据前，建议先重新获取通行证。',buttons:[{l:'重新拿通行证',f:'resetAuthAndStart()'},{l:'先看实时排队',f:"go('qt')"}]};
 else if(!hasStores)plan={level:'warn',mode:'还差门店',title:'通行证好了，下一步选门店',copy:'选好常用门店后，排队、预测、可约日历和自动抢预约都会自动带入，不用每页重选。',buttons:[{l:'设置门店和偏好',f:'openSnPrefs()'},{l:'先看实时排队',f:"go('qt')"}]};
 else plan={level:'ok',mode:'准备就绪',title:'可以开始未来预约或取号',copy:'通行证和门店偏好都已就绪。先查未来可约日历，目标明确再交给自动抢预约。',buttons:[{l:'查可约时段',f:"enterAdvanced('ca')"},{l:'自动抢预约',f:"enterAdvanced('sn')"}]};
 const steps=[
  journeyStepHTML('read','只读','排队、预测、叫号估算，直接用','ok'),
  journeyStepHTML('auth','通行证',authDesc,hc?(stale?'bad':'ok'):'warn'),
  journeyStepHTML('action','会执行',actionDesc,es.status==='error'?'bad':(running?'warn':(hc&&hasStores?'ok':'warn')))
 ];
 box.className='journey-panel mt16 '+plan.level;
 box.innerHTML='<div class="journey-head"><div><div class="journey-kicker">只读 / 通行证 / 会执行</div><h2>'+esc(plan.title)+'</h2><p class="journey-copy">'+esc(plan.copy)+'</p></div><span class="journey-mode '+escA(plan.level)+'">'+esc(plan.mode)+'</span></div><div class="journey-steps">'+steps.join('')+'</div><div class="journey-cta">'+journeyButtonsHTML(plan.buttons)+'</div>';
}
function openGuestStorePicker(){openStorePicker({selected:(pr.selected_stores||[]).map(String),onConfirm:saveStarterStores})}
async function saveStarterStores(ids){if(!ids||!ids.length){toast('先勾选至少一家门店');return}const b={...pr,selected_stores:ids,store_priority:ids};if(!await savePrefsPayload(b,true))return;qtSelected=ids.map(String);rememberStores('sushiro_qt_stores',qtSelected);toast('已记住常用门店，看看现在排多久');go('qt')}
let activeTickets=[],activeLive={},activeLoadedAt=0,homeLiveAt=0;
function lDA(){loadActiveTickets(false);loadHomeLive(false)}
function goQtStore(id){qtSelected=[String(id)];rememberStores('sushiro_qt_stores',qtSelected);go('qt')}
async function loadHomeLive(force){
 const box=el('homeLive');if(!box)return;
 const now=Date.now();if(!force&&now-homeLiveAt<60000)return;homeLiveAt=now;
 let all=recallStores('sushiro_qt_stores');if(!all.length)all=(pr.selected_stores||[]).map(String);
 const fallback=!all.length;if(fallback&&stores.length)all=stores.map(s=>String(s.id));
 const ids=all.slice(0,3);
 if(!ids.length){box.innerHTML=hc?'<div class="empty">还没选常用门店；选好后首页直接看排队。<div class="mt8"><button class="bt bt-w bt-s" onclick="openGuestStorePicker()">选常用门店</button></div></div>':'';return}
 if(!box.innerHTML)box.innerHTML='<div class="home-live">'+ids.map(()=>'<div class="hl-card"><span class="hl-name mu">读取中…</span></div>').join('')+'</div>';
 const panels=await Promise.all(ids.map(id=>safeFetch('/api/queue/live?store='+encodeURIComponent(id),null,12000).catch(()=>null)));
 const items=panels.filter(Boolean);
 if(!items.length){box.innerHTML='<div class="empty">实时排队读取失败。<div class="mt8"><button class="bt bt-w bt-s" onclick="loadHomeLive(true)">重试</button></div></div>';return}
 const more=all.length>ids.length?'<button type="button" class="hl-card" onclick="go(\'qt\')"><span class="hl-name">还有 '+(all.length-ids.length)+' 家关注门店</span><span class="hl-num">→</span><span class="hl-sub">去「现在去吃」看全部</span></button>':'';
 box.innerHTML='<div class="home-live">'+items.map(s=>{
  const open=s.online_open||s.store_status==='OPEN';
  const eta=(s.eta_minutes!=null)?s.eta_minutes:((s.server_wait_minutes||0)>0?s.server_wait_minutes:null);
  return'<button type="button" class="hl-card" onclick="goQtStore(\''+escA(String(s.store_id))+'\')"><span class="hl-name">'+esc(s.store_name||s.store_id)+'</span><span class="hl-num '+(open?'':'closed')+'">'+(open?fmtN(s.wait_groups||0):'休')+'</span><span class="hl-sub">'+(open?('桌在等'+(eta!=null?' · 约 '+eta+' 分钟':'')+(s.called_no?' · 叫到 '+esc(String(s.called_no)):'')):'暂停营业 · 点开看详情')+'</span></button>'}).join('')+more+'</div>'+(fallback?'<p class="mu mt8">还没选常用门店，暂按通行证里你去过的门店显示。<button class="bt bt-w bt-s" onclick="openGuestStorePicker()">选常用门店</button></p>':'');
}
async function loadActiveTickets(force){
 if(!hc){activeTickets=[];renderActiveHome();return}
 const now=Date.now();
 if(!force&&now-activeLoadedAt<60000){renderActiveHome();return}
 activeLoadedAt=now;
 try{const d=await safeFetch('/api/reservations',null,15000);const items=(Array.isArray(d)?d:(d.items||[]));activeTickets=items.filter(r=>{const st=String(r.status||'').toUpperCase();return!/CANCEL|EXPIRE|FINISH|SEATED|DONE/.test(st)})}catch(e){activeTickets=[]}
 renderActiveHome();
 const seen=new Set();
 for(const r of activeTickets){
  if(recordKind(r)!=='net_ticket')continue;
  const id=String(r.monitored_store_id||r.storeId||'');
  if(!id||seen.has(id))continue;seen.add(id);
  try{activeLive[id]=await safeFetch('/api/queue/live?store='+encodeURIComponent(id),null,12000)}catch(e){}
 }
 if(seen.size)renderActiveHome();
}
function renderActiveHome(){
 const box=el('activeHome');if(!box)return;
 const list=activeTickets||[],show=hc&&list.length>0;
 box.innerHTML=show?list.map(ticketHeroHTML).join(''):'';
 const hero=el('heroBox');if(hero)hero.classList.toggle('hid',show);
 renderJourneyPanel();
}
function ticketHeroHTML(r){
 const kind=recordKind(r),storeId=String(r.monitored_store_id||r.storeId||''),store=r.store_name||storeDisplayName(storeId)||storeId;
 if(kind==='net_ticket'){
  const live=activeLive[storeId]||null,lines=[];
  if(live&&live.called_no)lines.push('现在叫到 '+esc(String(live.called_no)));
  if(r.wait>0)lines.push('你前面还有约 '+fmtN(r.wait)+' 桌');
  else if(live&&live.wait_groups>0)lines.push('店内在等约 '+fmtN(live.wait_groups)+' 桌');
  if(live&&live.eta_minutes!=null)lines.push('约等待 '+live.eta_minutes+' 分钟');
  const no=String(r.number||'-');
 return'<div class="ticket-hero"><div class="th-eyebrow">🎫 你正在排：'+esc(store)+'</div><div class="th-no">'+esc(no)+'</div><div class="th-line">'+(lines.length?lines.join(' · '):'点下方按钮看“几点叫到我”')+'</div><div class="th-sub">'+esc(r.checkedIn?'已签到':'未签到')+' · 进度以寿司郎小程序为准</div><div class="th-acts"><button class="bt bt-w" onclick="openTicketForecast(\''+escA(storeId)+'\',\''+escA(no)+'\')">⏱ 几点叫到我 / 设提醒</button><button class="bt bt-ghost" onclick="enterAdvanced(\'re\')">查看单据</button><button class="bt bt-ghost advanced-only" onclick="cancelNetTicket()">取消排队号…</button></div></div>';
 }
 const when=r.slot_label||[r.queueDate,fT(r.start),r.end?'-'+fT(r.end):''].filter(Boolean).join(' ');
 return'<div class="ticket-hero"><div class="th-eyebrow">📅 你有一个预约：'+esc(store)+'</div><div class="th-no">'+esc(when||String(r.number||'-'))+'</div><div class="th-line">'+esc(recordStatusText(r,kind))+(r.number?' · 预约号 '+esc(String(r.number)):'')+'</div><div class="th-sub">预约号不参与当天叫号进度；到点前记得出发。</div><div class="th-acts"><button class="bt bt-w" onclick="enterAdvanced(\'re\')">查看单据</button><button class="bt bt-ghost" onclick="go(\'qt\')">看门店现场排队</button></div></div>';
}
function openTicketForecast(storeId,no){qdSelected=storeId?[String(storeId)]:[];rememberStores('sushiro_qd_store',qdSelected);const t=el('qdTargetNo'),n=parseInt(String(no||'').replace(/\D+/g,''),10);if(t)t.value=n>0?n:'';go('qd')}
function explainMsg(m){m=String(m||'');if(/机器级|LocalMachine|管理员权限|UAC|RunAs|elevated|exit code/i.test(m))return'Windows 机器级证书没装上：PC 微信只读机器级证书库，装它时会弹 UAC 请求管理员权限，点「是」即可。被拒或关掉就会失败——重新获取凭证会再弹一次，这次点同意。';if(/证书|trust|certificate/i.test(m))return'证书问题：先到设置页刷新诊断，确认 CA 证书已信任；失败后可重新获取凭证。';if(/代理|proxy/i.test(m))return'代理问题：先点击设置页的“修复代理”，再重新获取凭证。';if(/401|403|凭证|认证|token|auth/i.test(m))return'凭证过期：重新获取凭证参数后再启动。';if(/network|timeout|超时|不可达|connection/i.test(m))return'网络问题：确认网络可访问寿司郎接口，稍后重试。';if(/门店|store/i.test(m))return'门店配置问题：检查设置页的预约/取号门店是否仍在可用列表中。';return'先查看设置页本机诊断和日志，处理红色项后重试。'}
function wechatLightHTML(w){if(!w)return'';let cls,txt,btn='';if(w.restarted&&w.running){cls='ok';txt='PC 微信已重新打开 ✓ 请在寿司郎小程序里点一次排队或预约'}else if(w.restarted){cls='ok';txt='检测到 PC 微信已重启'}else if(w.running){cls='warn';txt='检测到 PC 微信正在运行——请彻底退出（任务栏右键退出，不是最小化）后重新打开';btn=' <button class="bt bt-o bt-s" onclick="killWeChat()">一键结束微信</button>'}else{cls='bad';txt='没检测到 PC 微信在运行，请打开 PC 微信'}return'<p class="wechat-light '+cls+'">'+esc(txt)+btn+'</p>'}
function uD(){
  const b=el('bm'),bc=el('bc'),nc=el('nc'),pick=el('heroPick'),title=el('heroTitle'),copy=el('heroCopy'),badge=el('heroBadge');
  const run=isRun();
  b.disabled=run;
  b.classList.remove('hid');
  bc.className='bt bt-w';
  bc.classList.remove('hid');
  bc.textContent='拿通行证';
  bc.onclick=startAuth;
  pick.classList.add('hid');
  nc.classList.add('hid');nc.textContent='';
  renderSetupCard();
  renderActiveHome();
  if(es.status==='capturing'){
    badge.textContent='正在捕获通行证';title.textContent='按引导操作一次小程序';copy.textContent='只需要点进寿司郎小程序产生一次请求，不要提交预约，也不要取消任何订单。抓到字段后下方进度会自动点亮。';
    b.textContent='捕获中';b.className='bt bt-y bt-l';b.onclick=sC;
    bc.classList.add('hid');
  }else if(es.status==='booking'||es.status==='sniping'){
    badge.textContent='正在执行';title.textContent=es.status==='sniping'?'蹲未来预约时段运行中':'自动抢预约运行中';copy.textContent=es.message||'页面可以保持打开；抢到未来预约后会保存记录、发送通知并停止。';
    b.textContent='运行中';b.className='bt bt-r bt-l';b.onclick=sB;
    bc.classList.add('hid');
  }else if(es.status==='success'){
    badge.textContent='已成功';title.textContent='已拿到预约 🍣';copy.textContent=es.message||'预约信息已保存。请以寿司郎小程序里的最终记录为准。';
    b.textContent='查看我的单据';b.className='bt bt-r bt-l';b.onclick=()=>enterAdvanced('re');
    bc.textContent='继续看排队';bc.onclick=()=>go('qt');
  }else if(es.status==='error'){
    badge.textContent='需要处理';title.textContent='运行遇到问题';copy.textContent='先看错误原因和建议。重新开始前，不会自动取消你的预约或排队号。';
    b.textContent=hc?'查看可约日历':'先看实时排队';b.className='bt bt-y bt-l';b.onclick=hc?(()=>enterAdvanced('ca')):(()=>go('qt'));
    bc.textContent=hc?'重新拿通行证':'拿通行证';
    bc.onclick=startAuth;
    nc.classList.remove('hid');nc.innerHTML='<b>错误</b><br><code style="word-break:break-all">'+esc(es.message||'(无错误信息)')+'</code><br><br><b>建议</b><br>'+esc(explainMsg(es.message));
  }else if(!hc){
    badge.textContent='第一次使用';title.textContent='想吃寿司郎？先看看现在排多久';copy.textContent='看门店、排队和叫号预测完全不需要通行证；只有抢未来预约、远程取号、读取我的单据才需要。';
    b.classList.add('hid');
    pick.classList.remove('hid');
    bc.textContent='我要抢预约：拿通行证（约 3 分钟）';
    bc.onclick=startAuth;
  }else{
    const hasStores=(pr.selected_stores||[]).length>0;
    if(!hasStores){
      badge.textContent='通行证已就绪';title.textContent='下一步：选门店和偏好';copy.textContent='抢未来预约前，需要先选门店、人数、桌型和时间偏好。只看排队仍然可以直接使用。';
      b.textContent='设置门店和偏好';b.className='bt bt-y bt-l';b.onclick=openSnPrefs;
      bc.textContent='先看实时排队';bc.onclick=()=>go('qt');
    }else{
      badge.textContent='准备就绪';title.textContent='今天怎么吃？';copy.textContent='通行证和门店偏好都已就绪。可以查未来可约日历直接预约；目标明确就交给自动抢预约。';
      b.textContent='查可约时段';b.className='bt bt-r bt-l';b.onclick=()=>enterAdvanced('ca');
      bc.textContent='自动抢预约';
      bc.className='bt bt-o';
      bc.onclick=()=>enterAdvanced('sn');
    }
  }
}
function uE(){
  const box=el('eb'),bs=el('bs'),s=es||{status:'idle'};
  if(!box){return}  // #eb 只在首页 DOM；SSE 在任意页面都可能触发 uE，元素不存在时直接跳过，避免 .classList 抛错中断后续状态更新
  const label={idle:'就绪',capturing:'正在捕获通行证',booking:'正在抢预约',sniping:'蹲预约中',success:'预约成功',error:'需要处理'}[s.status]||s.status;
  const desc=s.message||({idle:'等待下一步。',capturing:'等待小程序请求。',booking:'正在查询未来预约时段。',sniping:'蹲未来预约窗口运行中。',success:'已保存预约信息。',error:'请查看日志。'}[s.status]||'');
  box.className='engine '+s.status+(s.status==='idle'?' hid':'');box.innerHTML='<div class="row"><span class="dot"></span><strong>'+esc(label)+'</strong></div><p>'+esc(desc)+'</p>';
  if(s.status==='booking'&&s.attempts)box.innerHTML+='<p>已查询 '+s.attempts+' 次</p>';
  if(s.status==='error'&&s.message)box.innerHTML+='<p>'+esc(explainMsg(s.message))+'</p>';
  if(s.status==='capturing'&&s.capture&&s.capture.wechat&&(pf==='windows'||pf==='darwin'))box.innerHTML+=wechatLightHTML(s.capture.wechat);
  if(bs)bs.classList.toggle('hid',!isRun());
  const cb=el('cb');
  if(s.status==='capturing'&&s.capture){if(cb)cb.classList.remove('hid');rG(s.capture)}else if(s.status!=='capturing'){if(cb)cb.classList.add('hid')}
}
function remTab(t){const once=t==='once';el('remOnce').classList.toggle('hid',!once);el('remDaily').classList.toggle('hid',once);el('remTabOnce').classList.toggle('on',once);el('remTabDaily').classList.toggle('on',!once)}
function expandSnPrefs(){const t=el('snPrefsTime');if(t)t.open=true;const d=el('snPrefs');if(d){d.open=true;d.scrollIntoView({behavior:'smooth',block:'start'})}}
async function openSnPrefs(){await enterAdvanced('sn');if(cp==='sn')setTimeout(expandSnPrefs,80)}
async function openSettingsFold(id){if(currentUIMode()!=='advanced'&&ADVANCED_FOLDS.has(id)){await enterAdvanced('se');if(currentUIMode()!=='advanced')return}else go('se');setTimeout(()=>{const d=el(id);if(d){d.open=true;d.scrollIntoView({behavior:'smooth',block:'start'})}},80)}
function openDiagnostics(){openSettingsFold('fold-safe');setTimeout(()=>lD(),120)}
function focusNotifySettings(){go('se');setTimeout(()=>{const x=el('nf');if(x){x.scrollIntoView({behavior:'smooth',block:'center'});x.focus()}},60)}
function renderSettingsStatus(){
 const box=el('settingsStatus');if(!box)return;
 const stale=hc&&ah&&ah.status==='stale';
 const cloudConn=!!cloudAuth.connected,cloudCfg=!!cloudAuth.configured;
 const cloudBaseOK=!!cloudAuth.baseline_connected;
 const spOK=!!(spState&&(spState.running||spState.enabled||spState.sample_runs>0));
 const items=[
  {t:'寿司郎通行证 🎫',d:!hc?'看排队不需要；抢未来预约、远程取号、读单据才需要':stale?'可能已失效，建议重新获取':'已就绪；之后过期会自动提醒',s:!hc?'warn':stale?'bad':'ok',a:!hc?{l:'去获取',f:'openAuthWizard()'}:stale?{l:'重新认证',f:'resetAuthAndStart()'}:{l:'看我的单据',f:"enterAdvanced('re')"}},
  {t:'通知渠道',d:nfc?('已配置'+(notifyChannels.length?('：'+notifyChannels.join('、')):'')):'不配置就收不到叫号提醒和抢到通知',s:nfc?'ok':'warn',a:nfc?{l:'测试通知',f:"tN('all')"}:{l:'去配置',f:'focusNotifySettings()'}}
 ];
 if(currentUIMode()==='advanced'){
  items.push({t:'GitHub 线上基准',d:cloudBaseOK?('GitHub 已登录，线上数据库已验证，图表可叠加全国线上基准'):cloudConn?('GitHub 已登录，线上数据库待验证。验证前图表会继续优先用本机数据'):'登录后叫号预测可叠加全国线上基准（可选）',s:cloudBaseOK?'ok':'warn',a:cloudConn?{l:'退出',f:'logoutCloudAuth()'}:{l:'登录 GitHub',f:'startCloudLogin()'}});
  items.push({t:'预测数据',d:spOK?'采集中，“几点叫到”会越来越准':'公开曲线已默认记录；想更准可开启凭证态采集',s:spOK?'ok':'warn',a:{l:'配置',f:"openSettingsFold('fold-sm')"}});
 }
 box.innerHTML=healthStripHTML(items);
}
async function checkUpdate(){try{const u=await(await fetch('/api/update')).json(),b=el('updBox');if(!b)return;if(u.current_version==='dev'){b.classList.add('hid');return}if(u.update_available){b.classList.remove('hid');b.innerHTML='<h2>版本更新</h2><div class="ps"><b>'+esc(u.latest_version)+'</b><span class="line">当前 '+esc(u.current_version)+'</span></div><a class="bt bt-w bt-s mt16" href="'+escA(u.url||'#')+'" target="_blank">打开 Release</a>'}else b.classList.add('hid')}catch(e){}}
function rG(c){const cg=el('cg');if(!cg){return}cg.innerHTML=need.map(k=>'<div class="ci '+(c[k]?'ok':'')+'">'+fieldName(k)+'</div>').join('')}
function fieldName(k){return {x_app_code:'App Code',query_auth:'查询凭证',reservation_auth:'预约凭证',user_agent:'设备信息',referer:'小程序来源',wechat_id:'微信 ID',phone_number:'手机号',store_ids:'门店'}[k]||k}
async function sC(){try{const d=await(await fetch('/api/engine/capture',{method:'POST'})).json();if(d.error)toast(d.error);await loadStatus();}catch(e){toast('启动失败')}}
async function resetAuthOnly(ask){if(ask!==false){if(!await confirmDialog({title:'重置寿司郎认证？',body:'这会清除本机保存的寿司郎凭证，并停止未执行的自动取号计划；不会取消已经拿到的预约或排队号。\\n寿司郎凭证会过期，也可能被手机端登录顶掉。重置后需要重新获取凭证。',ok:'重置认证',cancel:'取消'}))return false}try{const d=await safeFetch('/api/auth/reset',{method:'POST'});hc=false;ah=d.auth_health||{status:'unknown'};await loadStatus();toast(d.message||'已重置认证');return true}catch(e){toast('重置认证失败：'+String(e.message||e));return false}}
async function resetAuthAndStart(){if(!await resetAuthOnly(true))return;openAuthWizard()}
async function rST(){if(!await confirmDialog('重置抓包状态？会断开当前抓包代理并清理残留，之后可点「获取凭证」手动重新连接。'))return;try{const d=await safeFetch('/api/engine/reset',{method:'POST'});if(d.error){toast(d.error);return}await loadStatus();toast('已重置抓包状态，点「获取凭证」可重新连接')}catch(e){toast('重置失败：'+String(e.message||e))}}
async function sB(){if(!await ensureNotifyConfigured('抢到预约'))return;if(!await confirmDialog('启动自动抢预约？\\n这会按你的门店和时段偏好尝试创建寿司郎预约；成功后会停止并保存到“我的单据”。\\n不会取消你已有的预约或排队号。'))return;try{const d=await(await fetch('/api/engine/booking',{method:'POST'})).json();if(d.error)toast(d.error);await loadStatus();}catch(e){toast('启动失败')}}
async function sE(){try{await fetch('/api/engine/stop',{method:'POST'});await loadStatus();}catch(e){}}
function startAuth(){if(hc&&(ah&&ah.status==='stale')){resetAuthAndStart();return}openAuthWizard()}
function mA(){hc?sB():startAuth()}
const MASCOT_KINDS=['salmon','maguro','saba','tamago','ebi','tako','unagi','hotate','ikura','uni','maki','kappa'];
function mascotFace(mood,fy){
 const my=fy+7;
 const eyes=mood==='sleep'?'<path d="M26 '+fy+'q3 3 6 0M40 '+fy+'q3 3 6 0" stroke="#3A3530" stroke-width="2.4" fill="none" stroke-linecap="round"/>':'<circle cx="29" cy="'+fy+'" r="2.6" fill="#3A3530"/><circle cx="43" cy="'+fy+'" r="2.6" fill="#3A3530"/>';
 const mouth=mood==='sad'?'<path d="M32 '+(my+2)+'q4 -3.5 8 0" stroke="#3A3530" stroke-width="2.2" fill="none" stroke-linecap="round"/>':mood==='happy'?'<path d="M32 '+my+'q4 4.5 8 0" stroke="#3A3530" stroke-width="2.2" fill="none" stroke-linecap="round"/>':'<path d="M33 '+(my+1)+'h6" stroke="#3A3530" stroke-width="2.2" stroke-linecap="round"/>';
 const blush=mood==='happy'?'<circle cx="23" cy="'+(fy+5)+'" r="2.4" fill="#F2A6A0" opacity=".75"/><circle cx="49" cy="'+(fy+5)+'" r="2.4" fill="#F2A6A0" opacity=".75"/>':'';
 return eyes+mouth+blush}
function mascotSVG(mood,size,kind){size=size||64;if(!kind||kind==='rand')kind=MASCOT_KINDS[Math.floor(Math.random()*MASCOT_KINDS.length)];
 const rice='<ellipse cx="36" cy="44" rx="27" ry="15" fill="#FFFDF6" stroke="#E5E0DB" stroke-width="2"/>';
 let body='';
 const topShape='M9 36Q36 12 63 36q1 6-5 7Q36 26 14 43q-6-1-5-7z';
 if(kind==='maki'){body='<circle cx="36" cy="32" r="27" fill="#33433A" stroke="#27332C" stroke-width="2"/><circle cx="36" cy="32" r="20" fill="#FFFDF6"/><circle cx="36" cy="23" r="6.5" fill="#F8875F"/><circle cx="28" cy="29" r="3.5" fill="#7FBF6C"/><circle cx="44" cy="29" r="3.5" fill="#FFD566"/>'+mascotFace(mood,37)}
 else if(kind==='kappa'){body='<circle cx="36" cy="32" r="27" fill="#33433A" stroke="#27332C" stroke-width="2"/><circle cx="36" cy="32" r="20" fill="#FFFDF6"/><circle cx="36" cy="23" r="7" fill="#6FB35D" stroke="#578F47" stroke-width="1.5"/><circle cx="36" cy="23" r="2.6" fill="#DFF0D6"/>'+mascotFace(mood,37)}
 else if(kind==='tamago'){body=rice+'<rect x="11" y="17" width="50" height="21" rx="10" fill="#FFD566" stroke="#E8B73F" stroke-width="2"/><rect x="31" y="13" width="10" height="30" rx="4" fill="#33433A"/>'+mascotFace(mood,41)}
 else if(kind==='ebi'){body=rice+'<path d="'+topShape+'" fill="#FB9C7C" stroke="#E27D5B" stroke-width="2" stroke-linejoin="round"/><path d="M24 32q4-4 8-5M36 26q5-1 9 0M48 27q5 1 8 4" stroke="#FFF1EA" stroke-width="3" fill="none" stroke-linecap="round"/>'+mascotFace(mood,41)}
 else if(kind==='maguro'){body=rice+'<path d="'+topShape+'" fill="#E8485C" stroke="#C9394B" stroke-width="2" stroke-linejoin="round"/><path d="M22 32q14 -9 28 0" stroke="#F8A8B2" stroke-width="2" fill="none" stroke-linecap="round"/>'+mascotFace(mood,41)}
 else if(kind==='unagi'){body=rice+'<path d="'+topShape+'" fill="#8C5A38" stroke="#6F4527" stroke-width="2" stroke-linejoin="round"/><path d="M20 33q7-5 14-6M42 26q7 0 12 4" stroke="#5C3A1F" stroke-width="2.5" fill="none" stroke-linecap="round"/><rect x="31" y="13" width="10" height="30" rx="4" fill="#33433A"/>'+mascotFace(mood,41)}
 else if(kind==='ikura'){body='<ellipse cx="36" cy="42" rx="24" ry="17" fill="#FFFDF6" stroke="#E5E0DB" stroke-width="2"/><rect x="12" y="18" width="48" height="22" rx="6" fill="#33433A" stroke="#27332C" stroke-width="2"/><circle cx="26" cy="15" r="6" fill="#FF9D45" stroke="#E8832E" stroke-width="1.5"/><circle cx="38" cy="12" r="6" fill="#FF9D45" stroke="#E8832E" stroke-width="1.5"/><circle cx="47" cy="16" r="6" fill="#FF9D45" stroke="#E8832E" stroke-width="1.5"/><circle cx="33" cy="17" r="5" fill="#FFB066" stroke="#E8832E" stroke-width="1.5"/>'+mascotFace(mood,47)}
 else if(kind==='uni'){body='<ellipse cx="36" cy="42" rx="24" ry="17" fill="#FFFDF6" stroke="#E5E0DB" stroke-width="2"/><rect x="12" y="18" width="48" height="22" rx="6" fill="#33433A" stroke="#27332C" stroke-width="2"/><path d="M20 10l-3-5M28 7l-1-5M37 6v-5M46 8l2-5M54 11l3-4" stroke="#B9842B" stroke-width="2" stroke-linecap="round"/><circle cx="26" cy="15" r="7" fill="#DFA63C" stroke="#C08A2D" stroke-width="1.5"/><circle cx="38" cy="12" r="7" fill="#E7B14A" stroke="#C08A2D" stroke-width="1.5"/><circle cx="49" cy="15" r="6" fill="#DFA63C" stroke="#C08A2D" stroke-width="1.5"/>'+mascotFace(mood,47)}
 else if(kind==='tako'){body=rice+'<path d="'+topShape+'" fill="#E89BB0" stroke="#C97891" stroke-width="2" stroke-linejoin="round"/><circle cx="27" cy="30" r="2.4" fill="#F8D8E1"/><circle cx="38" cy="27" r="2.4" fill="#F8D8E1"/><circle cx="48" cy="31" r="2.4" fill="#F8D8E1"/>'+mascotFace(mood,41)}
 else if(kind==='hotate'){body=rice+'<path d="'+topShape+'" fill="#F6E9D2" stroke="#DCC49C" stroke-width="2" stroke-linejoin="round"/><path d="M27 29q-1 5-2 9M36 26q0 6 0 12M45 29q1 5 2 9" stroke="#E3CFA8" stroke-width="2.5" fill="none" stroke-linecap="round"/>'+mascotFace(mood,41)}
 else if(kind==='saba'){body=rice+'<path d="'+topShape+'" fill="#AFC4D8" stroke="#85A0B8" stroke-width="2" stroke-linejoin="round"/><path d="M22 31q5-6 9-7M35 25q4-2 8-1M46 26q5 2 8 6" stroke="#5E7A93" stroke-width="2.4" fill="none" stroke-linecap="round"/>'+mascotFace(mood,41)}
 else{body=rice+'<path d="'+topShape+'" fill="#F8875F" stroke="#E0744C" stroke-width="2" stroke-linejoin="round"/><path d="M20 33q16 -10 32 0" stroke="#FFD9C9" stroke-width="2" fill="none" stroke-linecap="round"/>'+mascotFace(mood,41)}
 return '<svg class="mascot" width="'+size+'" height="'+size+'" viewBox="0 0 72 64" aria-hidden="true">'+body+'</svg>'}
function mascotRowHTML(mood,size){return '<div class="mascot-row">'+MASCOT_KINDS.map(k=>mascotSVG(mood,size||44,k)).join('')+'</div>'}
function fillPageMascots(){document.querySelectorAll('.pm').forEach(x=>{if(!x.innerHTML)x.innerHTML=mascotSVG(x.dataset.mood||'happy',x.dataset.size?+x.dataset.size:34,x.dataset.kind||'rand')})}
function buildBelt(){const b=el('belt');if(!b)return;
 // 无缝循环：轨道 = 完全相同的两段，translateX(-50%) 回到起点时画面逐像素一致。
 // 一段必须铺得比视口还宽，否则宽屏右侧会露出空白。itemW = 盘子 48 + 间距 56。
 const itemW=104,need=Math.max(window.innerWidth||1280,1280)+itemW;
 let half=[];while(half.length*itemW<need)half=half.concat(MASCOT_KINDS);
 const seg=half.map(k=>'<div class="belt-item">'+mascotSVG('plain',34,k)+'<i class="plate"></i></div>').join('');
 const dur=Math.round(half.length*itemW/26); // 恒定 ~26px/s，与宽度无关
 b.innerHTML='<div class="belt-track" style="animation-duration:'+dur+'s">'+seg+seg+'</div>'}
let beltResizeT=null;
window.addEventListener('resize',()=>{clearTimeout(beltResizeT);beltResizeT=setTimeout(buildBelt,400)});
function lsGet(k){try{return localStorage.getItem(k)||''}catch(e){return''}}
function lsSet(k,v){try{localStorage.setItem(k,v)}catch(e){}}
function rememberStores(k,ids){lsSet(k,(ids||[]).join(','))}
function recallStores(k){const v=lsGet(k);return v?v.split(',').filter(Boolean):[]}
function openFirstUseWizard(){let ov=el('firstUse');if(!ov){ov=document.createElement('div');ov.id='firstUse';ov.className='ov';document.body.appendChild(ov)}
 ov.innerHTML='<div class="ovc" style="width:min(720px,96vw)"><div class="fl ai jb mb16"><b>欢迎来吃寿司 🍣</b><button class="bt bt-w bt-s" onclick="closeFirstUseWizard()">跳过</button></div>'
 +'<div class="mascot-wrap">'+mascotRowHTML('happy',46)+'</div>'
 +'<h2 style="text-align:center;margin:4px 0 6px">想吃寿司郎？先看看现在排多久</h2>'
 +'<p class="mu" style="text-align:center;line-height:1.8">选一家你常去的店，马上看到实时排队——不用登录、不用通行证。<br>选过的店会被记住，之后看排队、叫号预测、约未来都自动带入。</p>'
 +'<div style="text-align:center;margin:16px 0 20px"><button class="bt bt-r bt-l" onclick="closeFirstUseWizard();openGuestStorePicker()">🔍 选我常去的门店</button></div>'
 +'<div class="task-grid">'
 +'<button class="task-card" type="button" onclick="firstUseGo(\'qd\',false)"><span class="tag read">只读 · 直接用</span><h3>我已经拿到号</h3><p>输入当天排队号，估几点叫到、几点出发。</p><div class="task-foot"><span class="mu">直接进入</span><span class="task-arrow">›</span></div></button>'
 +'<button class="task-card" type="button" onclick="firstUseGo(\'ca\',true)"><span class="tag auth">需要通行证 🎫</span><h3>想约未来某天</h3><p>看未来可约时段；需要时会先切到进阶版。</p><div class="task-foot"><span class="mu">没有通行证会先引导获取</span><span class="task-arrow">›</span></div></button>'
 +'</div></div>';
 ov.onclick=e=>{if(e.target===ov)closeFirstUseWizard()};
 ov.classList.remove('hid');ov.style.display='flex'}
function closeFirstUseWizard(){const ov=el('firstUse');if(ov){ov.classList.add('hid');ov.style.display='none'}}
async function firstUseGo(page,needsAuth){closeFirstUseWizard();if(needsAuth&&!hc){if(await confirmDialog({title:'需要先拿通行证',body:'这个功能需要先拿一次通行证（约 3 分钟）。\\n只看排队和叫号预测不用；抢未来预约或蹲未来预约时段才需要。\\n现在去拿？',ok:'去拿通行证',cancel:'先看看'}))startAuth();else go(page);return}go(page)}
let authWizPoll=null;
let awz={step:1,device:'',cap:null};
function awzSave(){try{localStorage.setItem('sushiro_wizard_state',JSON.stringify({step:awz.step,device:awz.device,cap:awz.cap}))}catch(e){}}
function awzClear(){awz={step:1,device:'',cap:null};try{localStorage.removeItem('sushiro_wizard_state');localStorage.removeItem('sushiro_wizard_draft')}catch(e){}}
function awzGo(n){awz.step=n;awzSave();authWizStep(n)}
function awzDevice(d){awz.device=d;awz.step=2;awzSave();authWizStep(2)}
function awzDraft(v){try{localStorage.setItem('sushiro_wizard_draft',v)}catch(e){}}
function awzStartPC(){closeAuthWizard();sC();go('da');toast('已启动 PC 微信自动捕获：打开 PC 微信里的寿司郎小程序，点一次门店，再点一次「我的预约」')}
function openAuthWizard(){let ov=el('authWiz');if(!ov){ov=document.createElement('div');ov.id='authWiz';ov.className='ov';document.body.appendChild(ov)}
 try{const s=JSON.parse(localStorage.getItem('sushiro_wizard_state')||'null');if(s&&s.step)awz={step:s.step,device:s.device||'',cap:s.cap||null}}catch(e){}
 if(awz.step>1&&awz.step<5&&!awz.device)awz.step=1;
 if(awz.step===5)awz.step=4;
 ov.classList.remove('hid');ov.style.display='flex';authWizStep(awz.step)}
function closeAuthWizard(){const ov=el('authWiz');if(ov){ov.classList.add('hid');ov.style.display='none'}if(authWizPoll){clearInterval(authWizPoll);authWizPoll=null}fetch('/api/mobile-auth/stop',{method:'POST',headers:{'X-Sushiro-CSRF':csrfToken}}).catch(()=>{})}
const AWZ_STEPS=['选设备','抓一次','传到电脑','粘贴解析','验证'];
function awzBar(cur){return'<div class="wsteps">'+AWZ_STEPS.map((s,i)=>{const n=i+1;return'<div class="wstep '+(n<cur?'done':n===cur?'on':'')+'"><i>'+(n<cur?'✓':n)+'</i>'+s+'</div>'}).join('')+'</div>'}
function authWizShell(cur,body){return'<div class="ovc"><div class="fl ai jb mb16"><b>拿通行证 🎫 <span class="mu" style="font-weight:400">约 3 分钟 · 全程只在本机处理</span></b><button class="bt bt-w bt-s" onclick="closeAuthWizard()">稍后再说</button></div>'+(cur?awzBar(cur):'')+'<div style="overflow:auto">'+body+'</div></div>'}
function awzChecklistHTML(){const c=awz.cap||{},got=need.filter(k=>c[k]).length;return'<div class="mu mt8" style="font-weight:800">字段捕获进度 '+got+'/'+need.length+'</div><div class="cg mt8">'+need.map(k=>'<div class="ci '+(c[k]?'ok':'')+'">'+(c[k]?'✓ ':'')+fieldName(k)+'</div>').join('')+'</div>'}
function awzToolHint(){return awz.device==='android'?'在手机上安装 <b>Reqable</b> / <b>HttpCanary</b> 等抓包工具，按它的引导装好并信任证书，然后开启抓包。':'在 App Store 安装 <b>Stream</b>（免费），按它的引导安装并信任证书，然后点「开始抓包」。'}
function authWizStep(step){const ov=el('authWiz');if(!ov)return;if(authWizPoll){clearInterval(authWizPoll);authWizPoll=null}
  if(step===1){
   const intro='<h3 style="margin:0 0 4px">第 1 步：怎么拿？</h3><p class="mu">通行证是寿司郎小程序和服务器对话用的身份凭证，抢预约、远程取号、读单据都靠它。原始字段只保存在本机，不会上传。</p>';
   const phones='<button class="bt bt-r" onclick="awzDevice(\'ios\')">📱 iPhone 手机抓包</button><button class="bt bt-r" onclick="awzDevice(\'android\')">🤖 安卓手机抓包</button>';
   const autoHint='<div class="why">💡 手机和电脑连同一个 Wi-Fi、路由器没开隔离？可以试 <button class="bt bt-w bt-s" onclick="authWizStep(\'auto\')">同 Wi-Fi 自动代理抓</button>，手机不用装抓包工具。</div>';
   const body=pf==='windows'
    ?intro+'<div class="wnum"><b class="n">!</b><div><b>Windows 上的 PC 微信抓不到小程序请求</b>，需要用手机拿一次，两条路任选：手机抓包（最稳），或同 Wi-Fi 自动代理。</div></div><div class="fl g8 fw mt16">'+phones+'</div>'+autoHint
    :intro+'<div class="fl g8 fw mt16"><button class="bt bt-r bt-l" onclick="awzStartPC()">💻 PC 微信自动抓（推荐 · 本机最省事）</button></div><p class="mu mt8">本机微信不方便？也可以用手机：</p><div class="fl g8 fw mt8">'+phones+'</div>'+autoHint;
   ov.innerHTML=authWizShell(1,body)}
  else if(step===2){ov.innerHTML=authWizShell(2,'<h3 style="margin:0 0 4px">第 2 步：在手机上“抓一次”</h3><p class="mu">'+awzToolHint()+'</p><div class="wnum"><b class="n">1</b><div>打开微信里的<b>寿司郎小程序</b></div></div><div class="wnum"><b class="n">2</b><div>随便点开一家门店 <span class="mu">← 这一下产生「查询请求」</span></div></div><div class="wnum"><b class="n">3</b><div>再点一次「<b>我的预约</b>」 <span class="mu">← 这一下产生「预约请求」</span></div></div><div class="why">💡 为什么要点两次？两类请求各含通行证的一半信息，缺一不可。</div><div class="fl ai jb mt16"><button class="bt bt-w bt-s" onclick="awzGo(1)">← 上一步</button><button class="bt bt-r" onclick="awzGo(3)">我点完了，下一步 →</button></div>')}
  else if(step===3){ov.innerHTML=authWizShell(3,'<h3 style="margin:0 0 4px">第 3 步：把抓到的内容传到电脑</h3><div class="wnum"><b class="n">1</b><div>在抓包工具里找到 <code>crm-cn-prd.sushiro.com.cn</code> 的请求（多选几条更稳）</div></div><div class="wnum"><b class="n">2</b><div>导出 / 复制成 <b>cURL</b> 或<b>原始请求头</b></div></div><div class="wnum"><b class="n">3</b><div>手机微信搜「<b>文件传输助手</b>」发给它 → 电脑微信打开同一会话复制</div></div><div class="why">💡 手机和电脑不在同一网络也没关系，文件传输助手走微信通道。</div><div class="fl ai jb mt16"><button class="bt bt-w bt-s" onclick="awzGo(2)">← 上一步</button><button class="bt bt-r" onclick="awzGo(4)">内容已复制，去粘贴 →</button></div>')}
  else if(step===4){let draft='';try{draft=localStorage.getItem('sushiro_wizard_draft')||''}catch(e){}
   ov.innerHTML=authWizShell(4,'<h3 style="margin:0 0 4px">第 4 步：粘贴并解析</h3><p class="mu">支持 JSON / cURL / 原始请求头。第一次没抓齐也没关系：<b>不要清空</b>，把新抓的内容接着粘在后面，再点一次解析。</p><div class="fg mt8"><label>抓包内容</label><textarea id="awImport" oninput="awzDraft(this.value)" placeholder="粘贴包含 X-App-Code、Authorization、User-Agent、Referer、wechatId、phoneNumber、storeId 的请求…"></textarea></div><div id="awChecklist">'+awzChecklistHTML()+'</div><div id="awImportState" class="diag-detail mt8 hid"></div><div class="fl ai jb mt16"><button class="bt bt-w bt-s" onclick="awzGo(3)">← 上一步</button><button class="bt bt-r" onclick="authWizImport()">解析并保存 →</button></div>');
   const ta=el('awImport');if(ta&&draft)ta.value=draft}
  else if(step===5){ov.innerHTML=authWizShell(5,'<div id="awVerify"></div>');awzVerify()}
  else if(step==='auto'){ov.innerHTML=authWizShell(0,'<h3 style="margin:0 0 4px">自动代理抓（同 Wi-Fi）</h3><p class="mu">手机不用装抓包工具：电脑临时帮手机“看一眼”寿司郎的网络请求（本机 MITM 代理，只解密寿司郎域名，其他流量不读取）。跟着信号灯走：</p><div id="awAutoStages"></div><div id="awAuto" class="mt8"><span class="mu">正在启动…</span></div><div class="fl g8 fw mt16"><button class="bt bt-w bt-s" onclick="awzGo(1)">← 返回选设备</button><button class="bt bt-w bt-s" onclick="closeAuthWizard()">停止并关闭</button></div>');authWizStartAuto()}}
async function authWizStartAuto(){try{const d=await safeFetch('/api/mobile-auth/start',{method:'POST'},12000);authWizRenderAuto(d);if(authWizPoll){clearInterval(authWizPoll);authWizPoll=null}authWizPoll=setInterval(authWizPollAuto,2500)}catch(e){const b=el('awAuto');if(b)b.innerHTML='<span class="bad">启动失败：'+esc(String(e.message||e))+'</span>'}}
async function authWizPollAuto(){try{const d=await safeFetch('/api/mobile-auth');authWizRenderAuto(d);if(d.saved||d.config_complete){if(authWizPoll){clearInterval(authWizPoll);authWizPoll=null}await loadStatus();toast('已捕获完成！记得把手机 Wi-Fi 代理改回关闭。');awz.step=5;awzSave();authWizStep(5)}}catch(e){}}
function awzAutoStages(d){const cap=d.capture||{},anyField=need.some(k=>cap[k]),done=!!(d.saved||d.config_complete);const st=[['电脑侧服务已启动，二维码可扫',!!d.active],['捕获到小程序请求',anyField],['字段齐全，已保存',done]];return st.map(x=>'<div class="strip"><span class="st '+(x[1]?'ok':'warn')+'">'+(x[1]?'✓':'…')+'</span><div><b>'+esc(x[0])+'</b></div></div>').join('')}
function authWizRenderAuto(d){const b=el('awAuto'),sg=el('awAutoStages');if(sg)sg.innerHTML=awzAutoStages(d);if(!b)return;const urls=d.guide_urls||[],hosts=d.hosts||[];b.innerHTML='<div class="wnum"><b class="n">1</b><div>手机微信「扫一扫」右侧二维码打开引导页，按页面提示<b>安装并信任 CA 证书</b>（iPhone 还需在 设置→通用→关于本机→证书信任设置 里完全信任）</div></div><div class="wnum"><b class="n">2</b><div>把手机 Wi-Fi 的 HTTP 代理设为下方 <b>电脑IP:端口</b></div></div><div class="wnum"><b class="n">3</b><div>彻底关掉再打开微信，进寿司郎小程序点一次门店，再点一次「我的预约」</div></div><div class="mt8" style="text-align:center">'+((d.active&&d.qr_svg)?d.qr_svg:'<span class="mu">二维码加载中…</span>')+'</div><div class="ps mt8">'+(urls.length?'<b>扫码或手机浏览器打开：</b><br>'+urls.map(u=>'<code>'+esc(u)+'</code>').join('<br>'):'')+'<div class="mu mt8"><b>Wi-Fi 代理：</b>'+hosts.map(h=>'<code>'+esc(h)+':'+esc(d.proxy_port||'')+'</code>').join(' ')+'</div><div class="mu mt8">扫码打不开 / 连不上？多半是路由器开了 AP（客户端）隔离，<button class="bt bt-w bt-s" onclick="awzDevice(awz.device||\'ios\')">改用手动抓（更稳）</button></div></div><div class="diag-detail mt8">'+esc(d.message||'')+'</div>'}
async function authWizImport(){const txt=(el('awImport')?.value||'').trim();if(!txt){toast('请先粘贴抓到的内容');return}const st=el('awImportState');if(st){st.classList.remove('hid');st.innerHTML='解析中…'}
 try{const d=await safeFetch('/api/auth/import',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({text:txt})},15000);
  const cap={};need.forEach(k=>{cap[k]=!!(d.capture&&d.capture[k])});awz.cap=cap;awzSave();
  const ck=el('awChecklist');if(ck)ck.innerHTML=awzChecklistHTML();
  if(d.saved){await loadStatus();awz.step=5;awzSave();authWizStep(5);return}
  const miss=d.missing||[],fix=[];
  if(miss.some(x=>/预约|微信|手机/.test(x)))fix.push('回到第 2 步，再点一次「我的预约」');
  if(miss.some(x=>/查询|Referer|门店/i.test(x)))fix.push('回到第 2 步，再点一次门店/排队');
  if(st)st.innerHTML='<span class="bad">还差一点，缺：</span>'+esc(miss.join('、')||'未知')+'<br><span class="mu">'+(fix.length?esc(fix.join('；'))+'，把新抓的内容接着粘在后面（不要清空），再点解析。':'再补一段包含缺失字段的请求，接着粘在后面即可。')+'</span>'
 }catch(e){if(st)st.innerHTML='<span class="bad">导入失败：'+esc(String(e.message||e))+'</span>'}}
function awzCelebrateHTML(){return'<div class="celebrate">'+mascotRowHTML('happy',50)+'<h3 style="margin:10px 0 4px;font-size:20px">通行证已生效！🍣</h3><p class="mu">抢预约、远程取号、读取我的单据都解锁了。原始凭证只保存在本机。</p><div class="fl g8 fw mt16" style="justify-content:center"><button class="bt bt-r" onclick="closeAuthWizard();enterAdvanced(\'ca\')">去约一个</button><button class="bt bt-w" onclick="closeAuthWizard();go(\'qt\')">先看排队</button><button class="bt bt-w bt-s" onclick="closeAuthWizard()">完成</button></div></div>'}
function awzConfetti(){const host=document.querySelector('#authWiz .celebrate');if(!host)return;const colors=['#B81C22','#F2A93B','#21823F','#F8875F','#4A90D9'];for(let i=0;i<16;i++){const s=document.createElement('span');s.className='confetti';s.style.left=(5+Math.random()*90)+'%';s.style.background=colors[i%colors.length];s.style.animationDelay=(Math.random()*0.5)+'s';host.appendChild(s);setTimeout(()=>s.remove(),2400)}}
async function awzVerify(){const box=el('awVerify');if(!box)return;box.innerHTML='<div class="mascot-wrap">'+mascotSVG('plain',64)+'</div><p class="mu" style="text-align:center">第 5 步：正在用通行证测试基础接口…</p>';
 try{const r=await fetch('/api/auth/probe',{method:'POST'}),d=await r.json();await loadStatus();
  if(d.ok){awzClear();box.innerHTML=awzCelebrateHTML();awzConfetti()}
  else{box.innerHTML='<div class="mascot-wrap">'+mascotSVG('sad',64)+'</div><div class="diag-detail bad">'+authProbeHTML(d)+'</div><div class="fl g8 fw mt16"><button class="bt bt-r bt-s" onclick="awzVerify()">重试</button><button class="bt bt-w bt-s" onclick="awzGo(4)">回到粘贴步骤</button></div>'}
 }catch(e){box.innerHTML='<div class="diag-detail bad">基础接口测试失败：'+esc(String(e.message||e))+'</div><div class="fl g8 fw mt16"><button class="bt bt-r bt-s" onclick="awzVerify()">重试</button><button class="bt bt-w bt-s" onclick="awzGo(4)">回到粘贴步骤</button></div>'}}

async function lC(){await ensureStores();if(!stores.length){el('storeChoices').innerHTML='<span class="mu">约未来需要先拿通行证 🎫；只看排队不用。</span>';el('sc').innerHTML='<div class="empty"><div class="mascot-wrap">'+mascotSVG('plain',56)+'</div>想查看未来可预约时段，需要先拿一次通行证（约 3 分钟）。只看实时排队请去「现在去吃」。<div class="mt8"><button class="bt bt-r bt-s" onclick="startAuth()">去拿通行证</button><button class="bt bt-w bt-s" onclick="go(\'qt\')">先看排队</button></div></div>';return}if(!selStores.length){el('sc').innerHTML='<div class="empty"><div class="mascot-wrap">'+mascotSVG('sleep',56)+'</div>还没选门店。选好后看看未来哪天有可约时段。<div class="mt8"><button class="bt bt-r bt-s" onclick="openStorePicker({selected:selStores,onConfirm:applyCalendarStores})">选择门店</button></div></div>';return}rStoreChoices();rC()}
function rStoreChoices(){const c=el('storeChoices');c.innerHTML=stores.map(s=>'<button class="chip '+(selStores.includes(String(s.id))?'on':'')+'" data-store="'+escA(String(s.id))+'">'+esc(s.nickname||s.name||s.id)+'</button>').join('');c.querySelectorAll('.chip').forEach(b=>b.onclick=()=>togStore(b.dataset.store))}
function togStore(id){selStores=selStores.includes(id)?selStores.filter(x=>x!==id):selStores.concat(id);if(!selStores.length&&stores[0])selStores=[String(stores[0].id)];rStoreChoices();sd='';rC()}
async function rC(){if(!selStores.length)return;el('sc').innerHTML='<div class="empty">加载中…</div>';const q='stores='+encodeURIComponent(selStores.join(','))+'&available='+(el('avOnly').checked?'1':'0')+'&period='+encodeURIComponent(el('period').value||'all');try{const d=await safeFetch('/api/calendar?'+q);if(d.error){el('sc').innerHTML=loadErrBoxHTML(d.error,'rC()','日历');return}as=[];calErrs=[];(d.stores||[]).forEach(st=>{if(st.error)calErrs.push({store:st.store_name||st.store_id,error:st.error});(st.slots||[]).forEach(s=>as.push({...s,store_name:st.store_name,store_id:st.store_id}))});rDB()}catch(e){el('sc').innerHTML=loadErrBoxHTML(e,'rC()','日历')}}
function setAR(){if(arTimer){clearInterval(arTimer);arTimer=null}const sec=+el('ar').value||0;if(sec>0)arTimer=setInterval(()=>{if(cp==='ca')rC()},sec*1000)}
function fD(d){return parseInt(d.substring(4,6),10)+'/'+parseInt(d.substring(6,8),10)}
function fT(t){return t&&t.length>=4?t.substring(0,2)+':'+t.substring(2,4):t||''}
function nT(t){t=compactTime(t||'');return t.length===4?t+'00':t}
function slotMatchesPrefs(s){const dt=new Date(s.date.substring(0,4)+'-'+s.date.substring(4,6)+'-'+s.date.substring(6,8)),w=dt.getDay(),rs=w===6?(pr.saturday_slots||[]):w===0?(pr.sunday_slots||[]):(pr.weekday_slots||[]),st=nT(s.start),en=nT(s.end||s.start);return rs.some(r=>st>=nT(r.start)&&st<nT(r.end)&&en<=nT(r.end))}
function calendarErrHTML(){return calErrs.length?'<div class="errbox">'+calErrs.map(x=>'<b>'+esc(x.store)+'</b>：'+esc(x.error)).join('<br>')+'<div class="mt8"><button class="bt bt-o bt-s" onclick="startAuth()">重新拿通行证</button></div></div>':''}
function rDB(){const g={};as.forEach(s=>{if(!g[s.date])g[s.date]=[];g[s.date].push(s)});const ds=Object.keys(g).sort(),b=el('dbar');b.innerHTML='';if(!ds.length){el('sc').innerHTML=calendarErrHTML()+'<div class="empty"><div class="mascot-wrap">'+mascotSVG('sleep',56)+'</div>这几家门店当前没有放出可展示时段，晚点再来看看？也可以刷新或换一家门店。</div>';return}if(!sd||!ds.includes(sd))sd=ds[0];ds.forEach(d=>{const sl=g[d],av=sl.filter(s=>s.availability==='AVAILABLE').length,dt=new Date(d.substring(0,4)+'-'+d.substring(4,6)+'-'+d.substring(6,8)),c=document.createElement('div');c.className='dc'+(d===sd?' on':'');c.innerHTML='<div class="dw">周'+W[dt.getDay()]+'</div><div class="dd">'+fD(d)+'</div><div class="dv '+(av>0?'h':'n')+'">'+(av>0?'可约 '+av:'已满')+'</div>';c.onclick=()=>{sd=d;rDB()};b.appendChild(c)});rS(sd)}
function rS(d){const sl=as.filter(s=>s.date===d).sort((a,b)=>(a.store_name||'').localeCompare(b.store_name||'')||(a.start||'').localeCompare(b.start||'')),c=el('sc');if(!sl.length){c.innerHTML=calendarErrHTML()+'<div class="empty">无时段</div>';return}const ac=sl.filter(s=>s.availability==='AVAILABLE').length;c.innerHTML=calendarErrHTML()+'<div class="sg">'+sl.map(s=>{const a=s.availability==='AVAILABLE',m=slotMatchesPrefs(s);return'<div class="sl '+(a?'av':'fu')+'"><div class="tm">'+esc(fT(s.start))+'-'+esc(fT(s.end))+'</div><div class="ss">'+(a?'可预约':'已满')+' · '+esc(s.store_name||s.store_id||'')+(a&&m?' · 符合偏好':'')+'</div><div class="mt8">'+(a?'<button class="bt bt-r bt-s" onclick="bookSlotDirect(\''+escA(String(s.store_id||''))+'\',\''+escA(s.date)+'\',\''+escA(s.start)+'\',\''+escA(s.end||'')+'\',\''+escA(String(s.store_name||s.store_id||''))+'\');return false">预约这个时段</button>':'<button class="bt bt-w bt-s" onclick="snFromSlot(\''+escA(String(s.store_id||''))+'\',\''+escA(s.date)+'\',\''+escA(s.start)+'\',\''+escA(s.end||'')+'\');return false">蹲这个时段</button>')+'</div></div>'}).join('')+'</div><p class="mu mt8">'+sl.length+' 个时段 · '+ac+' 个可预约（可直接预约）· 已满时段可加入蹲未来预约 · '+selStores.length+' 家门店</p>'}

async function lI(){await ensureStores();const c=el('ic');c.innerHTML='<div class="skeleton" style="height:46px;border-radius:10px;margin-bottom:8px"></div><div class="skeleton" style="height:200px;border-radius:10px"></div>';try{const d=await safeFetch('/api/insights?top=12');if(d.error){c.innerHTML=loadErrBoxHTML(d.error,'lI()','历史洞察');return}const rec=d.recommendations||[],min=d.min_recommendation_observations||3;const metrics='<div class="metric">'+chip('历史样本',d.valid_snapshots||0,'ok')+chip('推荐门槛','同一时段 '+min+' 次','warn')+chip('推荐数量',rec.length,'ok')+'</div>';const rows=rec.map(r=>'<tr><td data-label="门店">'+esc(storeName(r.store_id))+'<span class="mu debug-only"><br>'+esc(r.store_id)+'</span></td><td data-label="星期">'+esc(r.weekday_name)+'</td><td data-label="时段">'+esc(fT(r.start))+'-'+esc(fT(r.end))+'</td><td data-label="开放概率">'+Math.round((r.availability_rate||0)*100)+'%</td><td data-label="售罄速度">'+(r.sold_out_minutes==null?'-':Math.round(r.sold_out_minutes)+' 分')+'</td><td data-label="样本">'+esc(r.observations)+'</td></tr>').join('');const empty=(d.valid_snapshots||0)?'<div class="empty">样本还不够稳定。保持预测准确度，等同一门店、星期、时段至少积累 '+min+' 次观察后再给推荐。<div class="mt8"><button class="bt bt-w bt-s" onclick="openSettingsFold(\'fold-sm\')">去预测准确度</button></div></div>':'<div class="empty">暂无历史数据。<div class="mt8"><button class="bt bt-w bt-s" onclick="openSettingsFold(\'fold-sm\')">去预测准确度</button></div></div>';c.innerHTML=metrics+(rows?'<table class="tbl tbl-cards"><thead><tr><th>门店</th><th>星期</th><th>时段</th><th>开放概率</th><th>售罄速度</th><th>样本</th></tr></thead><tbody>'+rows+'</tbody></table>':empty)}catch(e){c.innerHTML=loadErrBoxHTML(e,'lI()','历史洞察')}}

async function lQD(){await ensureStores();if(!qdSelected.length){const saved=recallStores('sushiro_qd_store').slice(0,1);if(saved.length)qdSelected=saved}renderDashboardStores();applyPlanDir();fillNetTicketStores();loadNetTicketRoutine();await loadCloudAuth(false);await loadSampling();await loadQueueAlerts();await loadQueueAlertStatus();await loadQueueDashboard();runPlanCalc();stopQDAutoRefresh();qdAutoTimer=setInterval(()=>{if(document.hidden)return;loadQueueAdvisorCard()},45000)}
function dashboardParams(){const p=new URLSearchParams();p.set('scope',qdSelected.length?'local':'all');p.set('date_type',dashboardDateType());p.set('window','12');p.set('bucket','10');const target=parseInt(el('qdTargetNo')?.value||'',10);if(target>0)p.set('target_no',String(target));if(qdSelected.length)p.set('stores',qdSelected.slice(0,1).join(','));return p}
function dashboardDateType(){const v=el('qdDateType')?.value||'all';return['all','weekday','weekend','holiday'].includes(v)?v:'all'}
function applyDashboardStores(ids){qdSelected=(ids||[]).slice(0,1).map(String);rememberStores('sushiro_qd_store',qdSelected);renderDashboardStores();renderReminderTemplateHint();loadQueueDashboard();loadQueueAlertStatus();runPlanCalcDebounced()}
function renderDashboardStores(){const c=el('qdStores');if(!c)return;if(!qdSelected.length){const target=parseInt(el('qdTargetNo')?.value||'',10);c.innerHTML='<span class="mu">'+(target>0?'已填写当天排队号：请先选择门店，避免用其他门店曲线误判。':'未指定门店：可先浏览样本最多、最新的门店；填当天排队号前建议选定门店。')+'</span>';renderTicketReminderCard();return}c.innerHTML=qdSelected.map(id=>'<button class="chip on" data-store="'+escA(String(id))+'">'+esc(storeDisplayName(id))+' ✕</button>').join('');c.querySelectorAll('.chip.on').forEach(b=>b.onclick=()=>{const id=b.dataset.store;qdSelected=qdSelected.filter(x=>x!==id);rememberStores('sushiro_qd_store',qdSelected);renderDashboardStores();renderReminderTemplateHint();loadQueueDashboard();loadQueueAlertStatus()})}
function qdReminderStore(){const id=qdSelected[0];if(!id)return null;return{id:String(id),name:storeDisplayName(id)}}
function reminderTemplatePoints(target,tpl){const presets={normal:[80,50,25],conservative:[120,90,60,30],urgent:[50,25,10]},offsets=presets[tpl]||[];return Array.from(new Set(offsets.map(n=>target-n).filter(n=>n>0&&n<=target))).sort((a,b)=>a-b)}
function reminderPointsFromInputs(target){const custom=alertNoList(el('qdrPoints')?.value||'').filter(n=>n<=target);if(custom.length)return custom.sort((a,b)=>a-b);return reminderTemplatePoints(target,el('qdrTemplate')?.value||'normal')}
function renderReminderTemplateHint(){const target=parseInt(el('qdTargetNo')?.value||'',10),input=el('qdrPoints'),tpl=el('qdrTemplate')?.value||'normal';if(input&&!(input.value||'').trim()){const pts=target>0?reminderTemplatePoints(target,tpl):[];input.placeholder=pts.length?'默认 '+pts.join(','):'如 1000,1025,1050'}renderDashboardStores();renderTicketReminderCard()}
function qaRuleThreshold(r){return(r&&r.notify_at_no)||(((r&&r.target_no)||0)-((r&&r.lead_groups)||0))||0}
function qaRuleKey(r){r=r||{};return r.type==='called_reach'?[r.store_id,r.type,r.wait_minutes||0,r.target_no||0,qaRuleThreshold(r)].join('|'):[r.store_id,r.type,r.wait_minutes||0,r.target_no||0].join('|')}
async function loadQueueAlertStatus(){try{qaStatus=await safeFetch('/api/queue/alerts/status');renderTicketReminderCard()}catch(e){renderTicketReminderCard('提醒状态加载失败：'+String(e.message||e))}}
function renderTicketReminderCard(err){
 const box=el('qdReminderStatus');if(!box)return;
 const nb=el('qdrNotifyBtn');if(nb)nb.textContent=nfc?'管理通知':'设置通知';
 if(err){box.innerHTML='<div class="ci bad">'+esc(err)+'</div>';return}
 const s=qdReminderStore(),target=parseInt(el('qdTargetNo')?.value||'',10),points=target>0?reminderPointsFromInputs(target):[],n=qaStatus.notifications||{},sampling=qaStatus.sampling||{},channels=(n.channels||[]).join('、')||'未配置',notifyClass=n.configured?'ok':'bad',sampleClass=sampling.running||sampling.daemon_running||sampling.system_auto_start?.enabled?'ok':'warn',hint=!s?'先在上方选一家门店（提醒只盯这家店的叫号）。':!target?'在上方「当天排队号」填你的号码，提醒会按节奏自动生成。':points.length?('将为 '+esc(s.name)+' · 当天排队号 '+fmtN(target)+'，在叫到 '+points.map(fmtN).join('、')+' 号时各提醒一次。'):'自定义号码无效：提醒号必须小于你的当天排队号。';
 const chips=chip('通知',channels,notifyClass)+chip('采集',sampling.running?'运行中':sampling.daemon_running?'后台运行':sampling.system_auto_start?.enabled?'已设开机采集':(sampling.message||'未持续采集'),sampleClass);
 const rules=(qaStatus.rules||[]).filter(x=>x.rule&&x.rule.type==='called_reach'&&(!s||String(x.rule.store_id)===s.id)&&(!target||x.rule.target_no===target));
 const rows=rules.length?'<div class="sg mt8">'+rules.map(x=>{
  const r=x.rule||{},cls=x.status==='fired'?'av':x.status==='due'?'fu':'av',eta=x.estimate_to_threshold_minutes!=null?(' · 预计 '+fmtN(x.estimate_to_threshold_minutes)+' 分钟到提醒点'):'',next=x.next?' · 下一条':'',key=x.key||qaRuleKey(r);
  return'<div class="sl '+cls+'"><div class="fl ai jb g8"><div class="ss">'+esc(x.label||r.label||((r.target_no||0)+'号'))+' · '+fmtN(r.target_no||0)+'号</div><button class="bt bt-o bt-s" onclick="removeQueueAlertByKey(\''+escA(key)+'\')">删除</button></div><div class="mu mt8">到/过 '+fmtN(x.threshold||qaRuleThreshold(r))+' 号提醒 · 当前 '+fmtN(x.current_called_no||0)+' · 还差 '+fmtN(x.remaining_to_threshold||0)+' 号'+eta+next+'</div><div class="mu mt8">'+esc(x.status_text||'监控中')+(r.travel_minutes?(' · 路程约 '+fmtN(r.travel_minutes)+' 分钟'):'')+' · 命中后自动删除</div></div>'
 }).join('')+'</div>':'';
 box.innerHTML='<div class="fl g8 fw">'+chips+'</div><div class="mu mt8">'+hint+'</div>'+rows
}
function reminderSamplingActive(){const s=(qaStatus&&qaStatus.sampling)||{};return !!(s.running||s.daemon_running||s.system_auto_start?.enabled)}
async function ensureTicketReminderSampling(storeID){if(!hc)return '';try{if(!spCfg||!Object.keys(spCfg).length)await loadSampling();const active=reminderSamplingActive(),id=String(storeID),ids=(spCfg.store_ids||[]).map(String),hasStore=ids.includes(id),nextIDs=Array.from(new Set([id].concat(ids)));if(active&&hasStore)return '';const payload={...spCfg,enabled:true,auto_start:true,interval_seconds:spCfg.interval_seconds||300,active_start:spCfg.active_start||'100000',active_end:spCfg.active_end||'220000',store_ids:nextIDs,use_preference_stores:false};let d=await safeFetch('/api/sampling',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});spCfg=d.config||payload;spState=d.state||spState;if(active){await loadSampling();return 'updated'}d=await safeFetch('/api/sampling/start',{method:'POST'});spState=d.state||spState;await loadSampling();return 'started'}catch(e){toast('提醒已保存，但配置本机采集失败：'+String(e.message||e));return ''}}
async function createTicketReminder(){const s=qdReminderStore();if(!s){toast('请先选门店');return}const target=parseInt(el('qdTargetNo')?.value||'',10);if(!target){toast('请填写当天排队号');return}const points=reminderPointsFromInputs(target);if(!points.length){toast('请填写有效提醒点，且不能大于当天排队号');return}const label=(el('qdrLabel')?.value||'').trim(),travel=Math.max(0,parseInt(el('qdrTravel')?.value||'',10)||0),tpl=el('qdrTemplate')?.value||'normal';try{let base=qtAlerts||[];try{const d=await safeFetch('/api/queue/alerts');base=(d&&d.rules)||base}catch(e){}const rules=base.filter(r=>!(String(r.store_id)===s.id&&r.type==='called_reach'&&Number(r.target_no||0)===target));points.forEach(n=>rules.push({store_id:s.id,store_name:s.name,label:label,type:'called_reach',target_no:target,notify_at_no:n,lead_groups:Math.max(0,target-n),travel_minutes:travel,template:tpl,enabled:true}));const saved=await safeFetch('/api/queue/alerts',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({rules:rules})});qtAlerts=(saved&&saved.rules)||rules;const samplingAction=await ensureTicketReminderSampling(s.id);await loadQueueAlertStatus();let msg='已生成 '+points.length+' 个提醒点';if(samplingAction==='started')msg+='，已启动本机采集';if(samplingAction==='updated')msg+='，已加入本机采集门店';if(!reminderSamplingActive())msg+='，需要先获取凭证并开启本机采集才会推送';toast(msg);if(!nfc){const goCfg=await confirmDialog({title:'提醒已生成，但还没配通知渠道',body:'提醒规则已保存，但通知渠道（飞书/Telegram/Bark/Server酱）没配的话，到点叫号不会推送给你——得自己盯着屏幕。现在去配一个？只需填一次。',ok:'去配置通知',cancel:'稍后'});if(goCfg)focusNotifySettings()}}catch(e){toast('生成提醒失败：'+String(e.message||e))}}
function renderDashboardSamplingCard(){
 const box=el('qdSamplingCard');if(!box)return;
 const s=spState||{},cfg=spCfg||{},q=spQueueState||{},running=!!s.running,enabled=!!(s.enabled||cfg.enabled);
 const cloudReady=!!(cloudAuth.baseline_connected||(qdDashboardData.baseline&&qdDashboardData.baseline.used));
 const cloudLoggedIn=!!cloudAuth.connected;
 const localNeedsAuth=!hc||q.needs_auth||q.auth_ok===false;
 const last=s.last_run_at?new Date(s.last_run_at).toLocaleString():'还没有',next=s.next_run_at?new Date(s.next_run_at).toLocaleString():'-',ids=(cfg.store_ids||[]).map(String),storeText=ids.length?ids.map(storeDisplayName).join('、'):'偏好门店';
 const msg=s.last_error||s.message||q.message||'开启后会在本机记录叫号、等位和可预约时段。';
 const toggle='<label class="switch"><input type="checkbox" '+(running?'checked':'')+' onchange="toggleDashboardSampling(this.checked)"> 本机持续采集</label>';
 const cloudButton=cloudReady||cloudLoggedIn?'<button class="bt bt-w bt-s" onclick="loadQueueDashboard()">刷新图表</button>':'<button class="bt bt-w bt-s" onclick="startCloudLogin()">登录 GitHub 获取线上基准</button>';
 const localActions=toggle+'<button class="bt bt-w bt-s" onclick="runDashboardSampleOnce()">收集一次</button>'+(running?'<button class="bt bt-o bt-s" onclick="stopSampling()">暂停</button>':'')+'<button class="bt bt-w bt-s" onclick="openSettingsFold(\'fold-sm\')">详细配置</button>';
 const actions=localNeedsAuth?cloudButton+'<button class="bt bt-o bt-s" onclick="startAuth()">小程序采集补强</button>':localActions;
 box.innerHTML='<div><p style="margin-top:0">图表走 GitHub + 线上数据库；小程序通行证只用于本机采集补强。只记录 '+esc(storeText)+' 的叫号、等位和可预约时段；本机采集数据只留在本机，不上传。</p><div class="sample-state">'+chip('图表',cloudReady?'线上基准可用':cloudLoggedIn?'GitHub 已登录，基准待验证':'登录 GitHub 获取线上基准',cloudReady?'ok':'warn')+chip('本机采集',running?'运行中':enabled?'已启用':'未启动',running?'ok':enabled?'warn':'')+chip('小程序通行证',localNeedsAuth?'采集需更新':'采集可用',localNeedsAuth?'warn':'ok')+chip('样本',s.queue_snapshots||s.snapshots||0,'ok')+chip('上次',last,'ok')+chip('下次',next,'ok')+chip('最近结果',msg,s.last_error?'warn':'ok')+'</div></div><div class="curve-sampling-actions">'+actions+'</div>'
}
async function toggleDashboardSampling(on){if(on&&!hc){toast('本机持续采集需要先拿通行证');renderDashboardSamplingCard();startAuth();return}try{if(!spCfg||!Object.keys(spCfg).length)await loadSampling();const ids=qdSelected.length?qdSelected.slice(0,1):(spCfg.store_ids||[]);const payload={...spCfg,enabled:!!on,auto_start:on?true:!!spCfg.auto_start,interval_seconds:spCfg.interval_seconds||300,active_start:spCfg.active_start||'100000',active_end:spCfg.active_end||'220000',store_ids:ids,use_preference_stores:ids.length===0};let d=await safeFetch('/api/sampling',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});spCfg=d.config||payload;spState=d.state||spState;if(on){d=await safeFetch('/api/sampling/start',{method:'POST'});spState=d.state||spState;toast('已启动本机持续采集')}else{d=await safeFetch('/api/sampling/stop',{method:'POST'});spState=d.state||spState;toast('已暂停本机持续采集')}await loadSampling();renderDashboardSamplingCard()}catch(e){toast('采集开关失败：'+String(e.message||e));await loadSampling();renderDashboardSamplingCard()}}
async function runDashboardSampleOnce(){if(!hc){toast('本机采集需要先拿通行证');startAuth();return}try{if(!spCfg||!Object.keys(spCfg).length)await loadSampling();const ids=qdSelected.length?qdSelected.slice(0,1):(spCfg.store_ids||[]);const payload={...spCfg,enabled:true,interval_seconds:spCfg.interval_seconds||300,active_start:spCfg.active_start||'100000',active_end:spCfg.active_end||'220000',store_ids:ids,use_preference_stores:ids.length===0};let d=await safeFetch('/api/sampling',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});spCfg=d.config||payload;spState=d.state||spState;d=await safeFetch('/api/sampling/once',{method:'POST'});spState=d.state||spState;const r=d.result||{};toast(r.skipped?'本轮跳过：'+(r.skip_reason||'未知原因'):'收集完成：'+(r.queue_snapshots||0)+' 条排队快照，'+(r.snapshots||0)+' 条时段');await loadSampling();renderDashboardSamplingCard()}catch(e){toast('收集失败：'+String(e.message||e));await loadSampling();renderDashboardSamplingCard()}}
async function loadQueueDashboard(){const adv=el('qdAdvisor');if(!adv)return;const token=++qdDashToken;qdRefreshToken++;adv.innerHTML='<div class="ci">正在生成到店建议…</div>';try{const d=await safeFetch('/api/queue/dashboard?'+dashboardParams().toString(),null,20000);if(token!==qdDashToken)return;qdDashboardData=d||{};renderQueueDashboard(d);renderDashboardSamplingCard()}catch(e){if(token!==qdDashToken)return;qdDashboardData={};adv.innerHTML=loadErrBoxHTML(e,'loadQueueDashboard()','到店建议');renderDashboardSamplingCard()}if(token===qdDashToken)loadQueueAdvisorCard()}

// ---------- 排队压力答案卡 + 压力主图（我有号码页顶部） ----------
function pressureClass(level){return 'press-'+(level||'unknown')}
async function loadQueueAdvisorCard(){const ans=el('qdAnswer'),pc=el('qdPressChart');if(!ans)return;const store=qdSelected[0]||'';if(!store){ans.innerHTML='<div class="ci">选一家门店，并填上你的当天排队号，这里直接给你「几点叫到、几点出发」。</div>';if(pc)renderPressureChart(pc,{points:[],message:'选门店后，这里把今天的叫号进度、排队压力和你的当天排队号画在同一张图上（未选门店时仍展示全国线上历史排队趋势）。'},null,0);return}const target=parseInt(el('qdTargetNo')?.value||'',10)||0,travel=Math.max(0,parseInt(el('qdrTravel')?.value||'',10)||0);const token=++qdRefreshToken;ans.innerHTML='<div class="ci">正在读取实时排队压力…</div>';let adv=null;try{const qs='store='+encodeURIComponent(store)+(target>0?'&target_no='+target:'')+(travel>0?'&travel_minutes='+travel:'');adv=await safeFetch('/api/queue/advisor?'+qs,null,15000);if(token!==qdRefreshToken)return;renderQueueAnswer(adv,target)}catch(e){if(token!==qdRefreshToken)return;ans.innerHTML=loadErrBoxHTML(e,'loadQueueAdvisorCard()','排队压力')}
 if(pc){try{const curve=await safeFetch('/api/queue/pressure/curve?store='+encodeURIComponent(store),null,20000);if(token!==qdRefreshToken)return;renderPressureChart(pc,curve,adv,target)}catch(e){if(token!==qdRefreshToken)return;pc.innerHTML=loadErrBoxHTML(e,'loadQueueAdvisorCard()','整合走势')}}}
function renderQueueAnswer(adv,target){const ans=el('qdAnswer');if(!ans)return;const cur=adv.current||{},p=adv.pressure||{},sp=adv.speed||{},eta=adv.eta||null,nfcOk=nfc;let lead='';if(eta&&eta.remaining_groups>0&&eta.wait_minutes_range){const wr=eta.wait_minutes_range,called=fmtN(cur.called_no||0),tip=eta.estimated_called_at_range?(shortTime(eta.estimated_called_at_range.early)+'-'+shortTime(eta.estimated_called_at_range.late)):shortTime(eta.estimated_called_at);lead='你的当天排队号是 '+fmtN(target)+'，当前叫到 '+called+'，预计 '+wr.low+'-'+wr.high+' 分钟后叫到（约 '+tip+'）。'+(eta.arrival_suggestion||'')}else if(eta&&eta.remaining_groups<=0){lead='你的当天排队号是 '+fmtN(target)+'，已经轮到或即将轮到，请尽快到店。'}else if(eta){lead=eta.arrival_suggestion||'实时和历史数据都不足，暂时无法预估叫到时间。'}else if(target>0){lead='当前叫到 '+fmtN(cur.called_no||0)+' 号，正在估算到你的时间…'}else{lead='当前叫到 '+fmtN(cur.called_no||0)+' 号，排队压力'+(p.label||'数据不足')+'。填上你的当天排队号，给你「几点叫到、几点出发」。'}const s15=sp.called_per_min_15!=null?(Math.round(sp.called_per_min_15*15)+' 桌'):'数据不足';const chips=[];chips.push(answerChip('当前叫到',fmtN(cur.called_no||0)||'-',''));if(eta&&eta.remaining_groups>0)chips.push(answerChip('还差',fmtN(eta.remaining_groups)+' 号',''));chips.push(answerChip('排队压力',p.label||'数据不足',pressureClass(p.level)));chips.push(answerChip('消化趋势',p.trend_label||'数据不足',''));chips.push(answerChip('近15分钟叫号',s15,''));if(eta&&eta.source_label)chips.push(answerChip('估算依据',eta.source_label,eta.source==='official'?'press-extreme':''));if(eta&&eta.estimated_called_at_range)chips.push(answerChip('预计叫到',shortTime(eta.estimated_called_at_range.early)+'-'+shortTime(eta.estimated_called_at_range.late),''));chips.push(answerChip('通知',nfcOk?'已配置':'未配置',nfcOk?'':'press-extreme'));const reason=p.reason?'<div class="mu mt8">'+esc(p.reason)+'</div>':'',sourceNote=(eta&&eta.source_note)?'<div class="mu mt8">'+esc(eta.source_note)+'</div>':'',warns=(adv.warnings||[]).length?'<div class="mu mt8" style="color:#c4561a">⚠ '+(adv.warnings||[]).map(esc).join('；')+'</div>':'';ans.innerHTML='<div class="answer-lead">'+esc(lead)+'</div><div class="answer-chips">'+chips.join('')+'</div>'+reason+sourceNote+warns}
function answerChip(label,value,cls){return '<div class="answer-chip"><span>'+esc(label)+'</span><strong class="'+(cls||'')+'">'+esc(String(value))+'</strong></div>'}
function hhmmMinute(t){const m=String(t||'').match(/^(\d{1,2}):(\d{2})/);return m?parseInt(m[1],10)*60+parseInt(m[2],10):null}
function historicalCalledPoints(d){return ((d&&d.called_curve)||[]).filter(p=>hhmmMinute(p.bucket)!=null&&(p.called_no_typical||0)>0).slice().sort((a,b)=>hhmmMinute(a.bucket)-hhmmMinute(b.bucket))}
function historicalQueueTrendPoints(d){return ((d&&d.trend)||[]).map(p=>({p:p,m:hhmmMinute(p.label)!=null?hhmmMinute(p.label):hhmmMinute(p.bucket)})).filter(o=>o.m!=null&&o.m>=600&&o.m<=1320&&(o.p.total_queue_groups||0)>0).sort((a,b)=>a.m-b.m).map(o=>Object.assign({},o.p,{_m:o.m}))}
function calledCurveSourceLabel(source){return source==='remote_baseline'?'线上数据库基准':source==='local'?'本机历史采样':'历史基准'}
function queueTrendSourceLabel(source){return source==='remote_baseline'?'线上数据库':source==='local'?'本机采样':'历史'}
function renderPressureChart(box,curve,adv,target){
 if(!box)return;
 const minM=600,maxM=1320;
 const points=(curve&&curve.points||[]).filter(p=>{const m=hhmmMinute(p.time);return m!=null&&m>=minM&&m<=maxM}).slice().sort((a,b)=>hhmmMinute(a.time)-hhmmMinute(b.time));
 const hist=historicalCalledPoints(qdDashboardData);
 const trend=historicalQueueTrendPoints(qdDashboardData);
 if(!points.length&&!hist.length&&!trend.length){box.innerHTML='<div class="empty">'+esc((curve&&curve.message)||'还没有今天的本机采样曲线。开启「本机持续采集」后会逐步补齐；现在可看上面的实时答案卡。')+'</div>';return}
 const trendMax=Math.max(1,...trend.map(t=>t.total_queue_groups||0));
 const calledMax=Math.max(10,...points.map(p=>p.called_no||0),...hist.flatMap(p=>[p.called_no_slow||0,p.called_no_typical||0,p.called_no_fast||0]),target>0?target:0);
 const w=1040,h=286,l=52,r=52,t=28,b=40,maxCalled=calledMax,x=m=>l+((m-minM))/(maxM-minM)*(w-l-r),yCall=v=>h-b-(v/maxCalled)*(h-t-b),yPress=s=>h-b-(Math.min(100,Math.max(0,s))/100)*(h-t-b);
 let svg='<svg viewBox="0 0 '+w+' '+h+'" preserveAspectRatio="xMidYMid meet" style="width:100%;aspect-ratio:'+w+'/'+h+'">';
 svg+='<text class="chart-axis-title" x="'+(l-6)+'" y="'+(t-8)+'" text-anchor="end">叫号</text><text class="chart-axis-title" x="'+(w-r+6)+'" y="'+(t-8)+'" text-anchor="start">压力</text>';
 for(let i=0;i<=4;i++){const yy=t+i*(h-t-b)/4,val=Math.round(maxCalled*(4-i)/4);svg+='<line class="chart-grid" x1="'+l+'" y1="'+yy+'" x2="'+(w-r)+'" y2="'+yy+'"></line><text class="chart-label" x="'+(l-8)+'" y="'+(yy+4)+'" text-anchor="end">'+fmtN(val)+(i===0?' 号':'')+'</text><text class="chart-label" x="'+(w-r+8)+'" y="'+(yy+4)+'" text-anchor="start" fill="#888">'+(100-25*i)+'</text>'}
 for(let hh=10;hh<=22;hh+=2){const xx=x(hh*60);svg+='<line class="chart-grid" x1="'+xx+'" y1="'+t+'" x2="'+xx+'" y2="'+(h-b)+'" opacity=".55"></line><text class="chart-label" x="'+xx+'" y="'+(h-9)+'" text-anchor="middle">'+(hh<10?'0':'')+hh+':00</text>'}
 svg+='<line class="chart-axis" x1="'+l+'" y1="'+(h-b)+'" x2="'+(w-r)+'" y2="'+(h-b)+'"></line>';
 const pressArea=points.map(p=>x(hhmmMinute(p.time))+','+yPress(p.pressure_score||0));
 if(pressArea.length){const base=(h-b);svg+='<polygon points="'+l+','+base+' '+pressArea.join(' ')+' '+(w-r)+','+base+'" fill="rgba(120,120,152,.18)" stroke="rgba(120,120,152,.5)" stroke-width="1"></polygon>'}
 const histPts=hist.map(p=>x(hhmmMinute(p.bucket))+','+yCall(p.called_no_typical||0));
 if(histPts.length>1)svg+='<polyline points="'+histPts.join(' ')+'" fill="none" stroke="var(--blue)" stroke-width="2.4" stroke-dasharray="7 5" stroke-linejoin="round" stroke-linecap="round" vector-effect="non-scaling-stroke"></polyline>';
 const yTrend=g=>h-b-(Math.min(trendMax,Math.max(0,g))/trendMax)*(h-t-b);
 const trendPts=trend.map(t=>x(t._m)+','+yTrend(t.total_queue_groups||0));
 if(trendPts.length>1)svg+='<polyline points="'+trendPts.join(' ')+'" fill="none" stroke="var(--green)" stroke-width="2.4" stroke-linejoin="round" stroke-linecap="round" stroke-dasharray="2 4" vector-effect="non-scaling-stroke"></polyline>';
 const callPts=points.filter(p=>(p.called_no||0)>0);
 let stepPath='';
 callPts.forEach((p,i)=>{const cx=x(hhmmMinute(p.time)),cy=yCall(p.called_no);stepPath+=(i===0?'M':'L')+cx+','+cy+' ';if(i<callPts.length-1){const nx=x(hhmmMinute(callPts[i+1].time));stepPath+='L'+nx+','+cy+' '}});
 if(stepPath)svg+='<path d="'+stepPath+'" fill="none" stroke="var(--red)" stroke-width="3" stroke-linejoin="round" stroke-linecap="round" vector-effect="non-scaling-stroke"></path>';
 else if(callPts.length===1){const cx=x(hhmmMinute(callPts[0].time)),cy=yCall(callPts[0].called_no);svg+='<circle cx="'+cx+'" cy="'+cy+'" r="5" fill="#B81C22" stroke="#fff" stroke-width="2"></circle>'}
 const nowMin=(()=>{const dd=new Date();return dd.getHours()*60+dd.getMinutes()})();
 if(nowMin>=minM&&nowMin<=maxM){const nx=x(nowMin);svg+='<line x1="'+nx+'" y1="'+t+'" x2="'+nx+'" y2="'+(h-b)+'" stroke="var(--red)" stroke-width="1.4" opacity=".8"></line><text class="chart-label" x="'+(nx+4)+'" y="'+(t+10)+'" fill="var(--red)">现在</text>'}
 else{svg+='<text class="chart-label" x="'+(l+6)+'" y="'+(t+12)+'" fill="#aaa">（非营业时间 10:00-22:00，图不更新）</text>'}
 if(target>0){const my=target<=maxCalled?yCall(target):t;svg+='<line x1="'+l+'" y1="'+my+'" x2="'+(w-r)+'" y2="'+my+'" stroke="var(--red)" stroke-width="1.4" stroke-dasharray="4 4" opacity=".9"></line><text class="chart-label" x="'+(w-r-4)+'" y="'+(my-4)+'" text-anchor="end" fill="var(--red)">'+(target>maxCalled?'我的当天排队号 '+fmtN(target)+'（较靠后）':'我的当天排队号 '+fmtN(target))+'</text>'}
 // 预测叫到区间带：有 ETA 时间区间时，在 x 轴对应时段画半透明绿带 + 顶部标注「预计几点叫到你」。
 const er=(adv&&adv.eta&&adv.eta.estimated_called_at_range)?adv.eta.estimated_called_at_range:null;
 if(er&&target>0){const eM=hhmmMinute(shortTime(er.early)),lM=hhmmMinute(shortTime(er.late));if(eM!=null&&lM!=null&&lM>=minM&&eM<=maxM){const xe=x(Math.max(minM,eM)),xl=x(Math.min(maxM,lM));if(xl>xe){svg+='<rect x="'+xe+'" y="'+t+'" width="'+(xl-xe)+'" height="'+(h-b-t)+'" fill="rgba(33,130,63,.14)"></rect>';svg+='<line x1="'+xe+'" y1="'+t+'" x2="'+xe+'" y2="'+(h-b)+'" stroke="var(--green)" stroke-width="1.2" stroke-dasharray="3 3" opacity=".7"></line><line x1="'+xl+'" y1="'+t+'" x2="'+xl+'" y2="'+(h-b)+'" stroke="var(--green)" stroke-width="1.2" stroke-dasharray="3 3" opacity=".7"></line>';const lab='预计 '+shortTime(er.early)+'-'+shortTime(er.late)+' 叫到你'+(adv.eta.remaining_groups>0?('（还差 '+fmtN(adv.eta.remaining_groups)+' 号）'):'');svg+='<text class="chart-label" x="'+((xe+xl)/2)+'" y="'+(t+12)+'" text-anchor="middle" fill="var(--green)" font-weight="900">'+esc(lab)+'</text>'}}}
 const etaTip=(adv&&adv.eta&&adv.eta.estimated_called_at_range)?('\n预计叫到你：'+shortTime(adv.eta.estimated_called_at_range.early)+'-'+shortTime(adv.eta.estimated_called_at_range.late)):'';
 hist.forEach(p=>{const cx=x(hhmmMinute(p.bucket)),cy=yCall(p.called_no_typical||0),tip=p.bucket+'\n历史典型叫到：'+fmtN(p.called_no_typical||0)+'\n保守/偏快：'+fmtN(p.called_no_slow||0)+' / '+fmtN(p.called_no_fast||0)+'\n样本：'+fmtN(p.sample_count||0)+' · '+fmtN(p.day_count||0)+' 天\n来源：'+calledCurveSourceLabel(p.source)+(p.confidence?'\n置信度：'+p.confidence:'');svg+='<g class="chart-hot" data-tip="'+escA(tip)+'" onmousemove="dashTip(event,this)" onclick="dashTip(event,this)" onmouseleave="hideDashTip()"><circle cx="'+cx+'" cy="'+cy+'" r="3" fill="#fff" stroke="var(--blue)" stroke-width="1.8"></circle></g>'});
 if(trendPts.length>1)trend.forEach(p=>{const cx=x(p._m),cy=yTrend(p.total_queue_groups||0),tip=(p.label||p.bucket)+'\n历史排队桌数：'+fmtN(p.total_queue_groups||0)+'\n历史等待：'+fmtN(p.total_wait_minutes||0)+' 分\n样本数：'+fmtN(p.sample_count||0)+'\n来源：'+queueTrendSourceLabel(p.source);svg+='<g class="chart-hot" data-tip="'+escA(tip)+'" onmousemove="dashTip(event,this)" onclick="dashTip(event,this)" onmouseleave="hideDashTip()"><circle cx="'+cx+'" cy="'+cy+'" r="2.6" fill="#fff" stroke="var(--green)" stroke-width="1.6"></circle></g>'});
 callPts.forEach((p,i)=>{const cx=x(hhmmMinute(p.time)),cy=yCall(p.called_no),s15=p.called_speed_15!=null?(Math.round(p.called_speed_15*15)+' 桌'):'数据不足',tip=p.time+'\n当前叫到：'+fmtN(p.called_no)+'\n排队压力：'+pressureLabelCN(p.pressure_level)+'\n等待桌数：'+fmtN(p.waiting_groups||0)+'\n官方等待：'+fmtN(p.official_wait_minutes||0)+' 分\n近15分钟叫号：'+s15+'\n来源：'+pressureSourceLabel(p.source)+(p.confidence?'\n置信度：'+p.confidence:'')+(i===callPts.length-1?etaTip:'');svg+='<g class="chart-hot" data-tip="'+escA(tip)+'" onmousemove="dashTip(event,this)" onclick="dashTip(event,this)" onmouseleave="hideDashTip()"><circle cx="'+cx+'" cy="'+cy+'" r="'+(i===callPts.length-1?5:3.5)+'" fill="'+(i===callPts.length-1?'#B81C22':'#fff')+'" stroke="#B81C22" stroke-width="2"></circle></g>'});
 svg+='</svg>';
 const notes=[];if(curve&&curve.message)notes.push(curve.message);if(hist.length&&qdDashboardData.called_summary&&qdDashboardData.called_summary.message)notes.push('历史推算线：'+qdDashboardData.called_summary.message);
 if(trendPts.length>1)notes.push('历史排队趋势：绿色虚线是 '+queueTrendSourceLabel(trend[0].source)+'的 '+(trend.length)+' 个时间窗的排队桌数走势（与叫号轴量纲不同，仅看高低）。');
 const note=notes.length?'<div class="mu mt8">'+esc(notes.join(' '))+'</div>':'';
 box.innerHTML=svg+'<div class="chart-legend"><span class="legend-line">今日叫号</span><span class="legend-history">历史叫号</span>'+(trendPts.length>1?'<span class="legend-turso-trend">排队桌数趋势</span>':'')+'<span class="legend-pressure">排队压力</span><span class="legend-now">现在</span><span class="legend-mine">我的号</span><span class="mu">10:00-22:00</span></div>'+note
}
function pressureSourceLabel(source){return {local:'本机采样',remote_latest:'线上最新',remote_baseline:'线上基准'}[source]||'未知'}
function pressureLabelCN(level){return {low:'低',medium:'中',high:'高',extreme:'极高'}[level]||'数据不足'}
function riskLabelCN(r){return {low:'风险低',medium:'风险中',high:'风险高'}[r]||'风险未知'}
function riskClass(r){return {low:'press-low',medium:'press-medium',high:'press-extreme'}[r]||'press-unknown'}
// ---------- 取号→几点吃 ----------
// 时间换算方向：pickup=几点取号→几点吃；meal=想几点吃→几点取号。用 localStorage 记忆，避免依赖被移除的 select。
function planDir(){try{return localStorage.getItem('sushiro_plan_dir')==='meal'?'meal':'pickup'}catch(e){return 'pickup'}}
function setPlanDir(d){try{localStorage.setItem('sushiro_plan_dir',d==='meal'?'meal':'pickup')}catch(e){}}
function applyPlanDir(){const d=planDir();el('qpPickupWrap').classList.toggle('hid',d!=='pickup');el('qwMealWrap').classList.toggle('hid',d!=='meal');el('qwTravelWrap').classList.toggle('hid',d!=='meal');const t=el('planTitle');const s=el('planSub');if(t)t.textContent=d==='meal'?'想几点吃 → 该几点取号':'几点取号 → 几点能吃上';if(s)s.textContent=d==='meal'?'填想吃的時間，算出建议取号时间（倒推，结果仅供参考）':'填取号时间，算出大概几点能吃上'}
function swapPlanDir(){setPlanDir(planDir()==='meal'?'pickup':'meal');applyPlanDir();runPlanCalcDebounced()}
let _planCalcTimer=null
function runPlanCalcDebounced(){clearTimeout(_planCalcTimer);_planCalcTimer=setTimeout(runPlanCalc,300)}
function onPlanDirChange(){applyPlanDir();runPlanCalcDebounced()}
function runPlanCalc(){planDir()==='meal'?loadQueueMealPlan():loadQueuePickupPlan()}
async function loadQueuePickupPlan(){const ans=el('qpAnswer');if(!ans)return;const store=qdSelected[0];if(!store){ans.innerHTML='<div class="ci">先在上方选一家门店。</div>';return}const pickup=(el('qpPickup')?.value||'').replace(':','');ans.innerHTML='<div class="ci">正在估算…</div>';try{const d=await safeFetch('/api/queue/plan?store='+encodeURIComponent(store)+'&pickup='+encodeURIComponent(pickup),null,15000);renderPickupPlan(d)}catch(e){ans.innerHTML=loadErrBoxHTML(e,'loadQueuePickupPlan()','取号规划')}}
function renderPickupPlan(d){const ans=el('qpAnswer');if(!ans)return;if(d.message&&!d.meal_range){ans.innerHTML='<div class="answer-lead">'+esc(d.message)+'</div>';return}const wr=d.wait_minutes_range||{},mr=d.meal_range||{},lead='如果 '+esc(d.pickup)+' 取号，预计 '+esc(mr.early||'?')+'-'+esc(mr.late||'?')+' 吃上（等待约 '+(wr.low||0)+'-'+(wr.high||0)+' 分钟）。';const chips=[answerChip('推荐就餐',esc((mr.early||'?')+'-'+(mr.late||'?')),''),answerChip('预计等待',(wr.low||0)+'-'+(wr.high||0)+' 分',''),answerChip('风险',riskLabelCN(d.risk),riskClass(d.risk))].join('');ans.innerHTML='<div class="answer-lead">'+lead+'</div><div class="answer-chips">'+chips+'</div>'+(d.basis?'<details class="plan-basis mt8"><summary>为什么</summary><div class="mu mt8">'+esc(d.basis)+'</div></details>':'')}
// ---------- 想几点吃→几点取号 ----------
async function loadQueueMealPlan(){const ans=el('qpAnswer');if(!ans)return;const store=qdSelected[0];if(!store){ans.innerHTML='<div class="ci">先在上方选一家门店。</div>';return}const meal=(el('qwMeal')?.value||'').replace(':',''),travel=Math.max(0,parseInt(el('qwTravel')?.value||'',10)||0);ans.innerHTML='<div class="ci">正在倒推…</div>';try{const d=await safeFetch('/api/queue/plan?store='+encodeURIComponent(store)+'&target_meal='+encodeURIComponent(meal)+(travel>0?'&travel_minutes='+travel:''),null,15000);renderMealPlan(d)}catch(e){ans.innerHTML=loadErrBoxHTML(e,'loadQueueMealPlan()','取号倒推')}}
function renderMealPlan(d){const ans=el('qpAnswer');if(!ans)return;if(d.message&&!d.recommend_pickup_range){ans.innerHTML='<div class="answer-lead">'+esc(d.message)+'</div>';return}const rp=d.recommend_pickup_range||{},wr=d.wait_minutes_range||{},lead='想 '+esc(d.target_meal)+' 吃，建议 '+esc(rp.early||d.stable_pickup||'?')+'-'+esc(rp.late||d.stable_pickup||'?')+' 取号。'+(d.latest_pickup?(' 最晚别拖过 '+esc(d.latest_pickup)+'。'):'');const chips=[answerChip('建议取号',esc((rp.early||'?')+'-'+(rp.late||'?')),''),answerChip('偏稳取号',esc(d.stable_pickup||'-'),''),answerChip('最晚取号',esc(d.latest_pickup||'-'),''),answerChip('预计等待',(wr.low||0)+'-'+(wr.high||0)+' 分',''),answerChip('风险',riskLabelCN(d.risk),riskClass(d.risk))].join('');ans.innerHTML='<div class="answer-lead">'+esc(lead)+'</div><div class="answer-chips">'+chips+'</div>'+(d.basis?'<details class="plan-basis mt8"><summary>为什么</summary><div class="mu mt8">'+esc(d.basis)+'</div></details>':'')+'<div class="mu mt8">⚠ 倒推按历史等待估的；取号后前面可能被插队，实际等待可能 ±15 分钟，别把建议取号时间当死线。</div>'}
function renderQueueDashboard(d){renderDashboardAdvisor(d.advisor||{});renderDashboardInsights(d);renderDashboardDataSource(d)}
function dashboardBaselineStatusHTML(d){const b=(d&&d.baseline)||{};const configured=!!b.configured,authenticated=!!b.authenticated,used=!!b.used,rollupCount=Number(b.rollup_count||0),latestCount=Number(b.latest_count||0),rollup=fmtN(rollupCount),latest=fmtN(latestCount);let title,lines=[],cls='ok';if(used){title='本次图表已使用线上数据库基准';lines.push('来源：线上数据库基准');if(rollupCount||latestCount)lines.push('聚合样本 '+rollup+' 条，最新明细 '+latest+' 条');else lines.push('基准已响应，暂无样本');cls='ok'}else if(authenticated){title='仍在用本机数据，线上数据库基准未参与本次图表';lines.push('GitHub 已登录，但线上数据库还没验证成功，需在设置页确认。');cls='warn'}else if(configured){title='本次图表用本机数据，GitHub 尚未登录';lines.push('云端服务已配置；登录 GitHub 后可验证线上基准并叠加参考。');cls='warn'}else{title='本次图表用本机数据，未配置线上基准';lines.push('可在「设置」登录 GitHub 并验证线上基准后，叠加全国线上参考。');cls='warn'}const ws=(d&&d.warnings)||[];if(ws.length){cls=cls==='ok'?'warn':cls;lines.push('注意：'+ws.join('；'));if(ws.some(w=>/明细|基准|曲线/.test(w)))lines.push('这能解释为什么基准可用、但叫号曲线仍没明细。')}return{cls:cls,html:'<b>📊 图表数据来源</b><p>'+esc(title)+'</p><div class="data-source-lines">'+lines.map(l=>'<span>'+esc(l)+'</span>').join('')+'</div>'}}
function renderDashboardDataSource(d){const box=el('qdDataSource');if(!box)return;const s=dashboardBaselineStatusHTML(d);box.className='data-source mt16 '+(s.cls||'');box.innerHTML=s.html||''}
function renderDashboardInsights(d){const heat=el('qdHeatmap'),wk=el('qdWeekday'),tr=el('qdTrend'),cc=el('qdCalledCurve');if(heat)heat.innerHTML=renderHeatmapHTML(d.heatmap||[]);
if(wk)wk.innerHTML=renderWeekdayHTML(d.weekday_profiles||[]);
if(tr)tr.innerHTML=renderTrendHTML(d.trend||[]);
if(cc)cc.innerHTML=renderCalledCurveHTML(d.called_curve||[])}
function fmtFloat(v,dft){return(v==null||isNaN(v))?dft:Math.round(Number(v))}
function busyClass(rate){if(rate==null)return'';const r=Number(rate);if(r>=0.65)return'hot';if(r>=0.35)return'warm';if(r>=0.15)return'mild';return''}
function renderHeatmapHTML(pts){if(!pts.length)return'<div class="empty">还没有这家店的热力图数据，开启本机采集后会逐步补齐。</div>';const days={};pts.forEach(p=>{days[p.weekday]=days[p.weekday]||{name:p.weekday_name,buckets:{}};days[p.weekday].buckets[p.bucket]=p});const wkOrder=[1,2,3,4,5,6,0];const buckets=[];pts.forEach(p=>{if(!buckets.includes(p.bucket))buckets.push(p.bucket)});buckets.sort();const head='<tr><th>时段</th>'+buckets.map(b=>'<th>'+esc(b)+'</th>').join('')+'</tr>';const rows=wkOrder.filter(w=>days[w]).map(w=>{const dn=days[w];return'<tr><td>'+esc(dn.name)+'</td>'+buckets.map(b=>{const p=dn.buckets[b];if(!p)return'<td></td>';const rate=p.busy_rate;const cls=busyClass(rate);const tip=(p.weekday_name||'')+' '+esc(b)+'\n平均等位 '+fmtFloat(p.wait_minutes_avg,'-')+' 桌数 '+fmtFloat(p.queue_groups_avg,'-')+'\n忙率 '+Math.round((Number(rate)||0)*100)+'% 样本 '+p.sample_count;return'<td title="'+escA(tip)+'"><span class="heat-cell '+cls+'">'+fmtFloat(p.queue_groups_avg,'-')+'</span></td>'}).join('')+'</tr>'}).join('');return'<p class="ph-sub" style="margin:0 0 8px">单元格数字=平均排队桌数，颜色越红代表该时段越忙；悬停看等位/忙率/样本。</p><div class="heat-wrap"><table class="heat"><thead>'+head+'</thead><tbody>'+rows+'</tbody></table></div>'}
function renderWeekdayHTML(profiles){if(!profiles.length)return'';const order=[1,2,3,4,5,6,0];const sorted=order.map(w=>profiles.find(p=>p.weekday===w)).filter(Boolean);return'<p class="ph-sub" style="margin:0 0 8px">工作日画像：平均排队桌数 / 平均等位 / 高峰时段。</p><div class="weekday-strip">'+sorted.map(p=>'<div class="weekday-card"><b>'+esc(p.weekday_name||'')+'</b><span>平均桌数 '+fmtFloat(p.queue_groups_avg,'-')+' · 等位 '+fmtFloat(p.wait_minutes_avg,'-')+' 分'+(p.peak_bucket?'<br>高峰约 '+esc(p.peak_bucket)+'（'+fmtFloat(p.peak_queue_groups,'-')+' 桌）':'')+'<br>样本 '+p.sample_count+' · '+esc(p.confidence||'')+'</span></div>').join('')+'</div>'}
function renderTrendHTML(trend){if(!trend.length)return'';const top=trend.slice(-12);const maxG=Math.max(1,...top.map(t=>t.total_queue_groups||0));return'<p class="ph-sub" style="margin:0 0 8px">近段排队趋势条（按采样窗口）。</p><div class="rank-list">'+top.map(t=>{const pct=Math.round((t.total_queue_groups||0)/maxG*100);return'<div class="rank-row"><b>'+esc(t.label||t.bucket)+'</b><span>桌数 '+fmtN(t.total_queue_groups)+' · 等位 '+fmtN(t.total_wait_minutes)+' 分 · 样本 '+t.sample_count+'</span><strong style="font-size:14px">'+fmtN(t.total_queue_groups)+'</strong></div>'}).join('')+'</div>'}
function renderCalledCurveHTML(curve){const pts=(curve||[]).filter(p=>hhmmMinute(p.bucket)!=null&&(p.called_no_typical||p.called_no_slow||p.called_no_fast)>0).slice().sort((a,b)=>hhmmMinute(a.bucket)-hhmmMinute(b.bucket));if(pts.length<2)return'<div class="empty">还没有这家店的历史叫号曲线。开启本机采集、或登录 GitHub 连上线上数据库后，这里会画出「按历史，几点大概叫到几号」的折线。</div>';const minM=600,maxM=1320;const inRange=pts.filter(p=>{const m=hhmmMinute(p.bucket);return m>=minM&&m<=maxM});const useP=inRange.length>=2?inRange:pts;const maxNo=Math.max(10,...useP.flatMap(p=>[p.called_no_typical||0,p.called_no_slow||0,p.called_no_fast||0]));const w=1000,h=260,l=54,r=24,t=24,b=38,x=m=>l+((m-minM))/(maxM-minM)*(w-l-r),y=v=>h-b-(Math.min(maxNo,v)/(maxNo))*(h-t-b);let svg='<svg viewBox="0 0 '+w+' '+h+'" preserveAspectRatio="xMidYMid meet" style="width:100%;aspect-ratio:'+w+'/'+h+'">';for(let i=0;i<=4;i++){const yy=t+i*(h-t-b)/4,val=Math.round(maxNo*(4-i)/4);svg+='<line class="chart-grid" x1="'+l+'" y1="'+yy+'" x2="'+(w-r)+'" y2="'+yy+'"></line><text class="chart-label" x="'+(l-8)+'" y="'+(yy+4)+'" text-anchor="end">'+fmtN(val)+(i===0?' 号':'')+'</text>'}for(let hh=10;hh<=22;hh+=2){const xx=x(hh*60);svg+='<line class="chart-grid" x1="'+xx+'" y1="'+t+'" x2="'+xx+'" y2="'+(h-b)+'" opacity=".55"></line><text class="chart-label" x="'+xx+'" y="'+(h-9)+'" text-anchor="middle">'+(hh<10?'0':'')+hh+':00</text>'}svg+='<line class="chart-axis" x1="'+l+'" y1="'+(h-b)+'" x2="'+(w-r)+'" y2="'+(h-b)+'"></line>';const poly=(key,color,dash)=>{const ps=useP.filter(p=>(p[key]||0)>0).map(p=>x(hhmmMinute(p.bucket))+','+y(p[key]));if(ps.length<2)return'';return'<polyline points="'+ps.join(' ')+'" fill="none" stroke="'+color+'" stroke-width="2.4" '+dash+' stroke-linejoin="round" stroke-linecap="round" vector-effect="non-scaling-stroke"></polyline>'};svg+=poly('called_no_slow','var(--blue)','stroke-dasharray="5 4"');svg+=poly('called_no_fast','#E0A800','stroke-dasharray="2 4"');svg+=poly('called_no_typical','var(--green)','');useP.forEach(p=>{const cx=x(hhmmMinute(p.bucket)),cy=y(p.called_no_typical||0),tip=escA((p.bucket||'')+'\n历史典型叫到：'+fmtN(p.called_no_typical||0)+' 号\n保守/偏快：'+fmtN(p.called_no_slow||0)+' / '+fmtN(p.called_no_fast||0)+' 号\n样本：'+fmtN(p.sample_count||0)+' · '+fmtN(p.day_count||0)+' 天');svg+='<g class="chart-hot" data-tip="'+tip+'" onmousemove="dashTip(event,this)" onclick="dashTip(event,this)" onmouseleave="hideDashTip()"><circle cx="'+cx+'" cy="'+cy+'" r="3" fill="#fff" stroke="var(--green)" stroke-width="1.8"></circle></g>'});svg+='</svg>';return'<p class="ph-sub" style="margin:0 0 8px">历史叫号曲线：按这家店的历史规律，每个时间点大概叫到几号。绿线=典型，蓝虚线=保守（慢），黄虚线=偏快；悬停看详情。</p><div class="chart">'+svg+'</div>'}
function renderDashboardAdvisor(a){const box=el('qdAdvisor');if(!box)return;a=a||{};const state=a.state||'empty',bad=state==='passed'||state==='empty',warn=state==='uncovered',cls=bad?'bad':warn?'warn':state==='milestones'?'muted':'';const source=a.source==='remote_baseline'?'线上基准':a.source?'本机记录':'无数据',conf=confText(a.confidence||'none'),target=a.target_no?('当天排队号 '+fmtN(a.target_no)):'未输入号码',miles=(a.milestones||[]).slice(0,3).map(m=>'<div class="advisor-point"><span>'+esc(m.label||'时间点')+'</span><b>'+esc(m.bucket||'-')+'</b><strong>'+fmtN(m.called_no_typical||0)+'号</strong></div>').join('');let side=miles||'<div class="advisor-point"><span>提示</span><b>选门店</b><strong>补数据</strong></div>';box.innerHTML='<div class="advisor-card '+cls+'"><div class="advisor-main"><span class="advisor-eyebrow">'+esc(target)+' · '+esc(source)+' · 可信度'+esc(conf)+'</span><h3>'+esc(a.headline||'还不能判断叫到时间')+'</h3><p>'+esc(a.copy||'先选一个门店；如果没有曲线，开启本机采集后会逐步变准。')+'</p>'+(a.arrival_label?'<p><b>到店建议：</b>'+esc(a.arrival_label)+'</p>':'')+'</div><div class="advisor-milestones">'+side+'</div></div>'}
function fmtN(v){return Number(v||0).toLocaleString('zh-CN')}
function trendDeltaText(v){return(v>0?'↑ '+fmtN(v):v<0?'↓ '+fmtN(Math.abs(v)):'平稳')}
function shortTime(v){if(!v)return'-';const d=new Date(v);if(Number.isNaN(d.getTime()))return String(v).slice(11,16)||String(v);return d.toLocaleTimeString('zh-CN',{hour:'2-digit',minute:'2-digit',hour12:false})}
function dashTip(e,node){let t=el('dashTip');if(!t){t=document.createElement('div');t.id='dashTip';t.className='dash-tip';document.body.appendChild(t)}t.textContent=node.getAttribute('data-tip')||'';t.style.display='block';const x=Math.min(window.innerWidth-280,e.clientX+14),y=Math.min(window.innerHeight-170,e.clientY+14);t.style.left=Math.max(8,x)+'px';t.style.top=Math.max(8,y)+'px'}
function hideDashTip(){const t=el('dashTip');if(t)t.style.display='none'}
async function lQT(){await ensureStores();initQueueTrendFilters();renderQueueTrendStores();await refreshQueueView()}
function initQueueTrendFilters(){if(!qtSelected.length)qtSelected=recallStores('sushiro_qt_stores');if(!qtSelected.length)qtSelected=(stores.length?stores.map(s=>String(s.id)):(pr.selected_stores||[]).map(String))}
function renderQueueTrendStores(){const c=el('qtStores');if(!c)return;if(!qtSelected.length){c.innerHTML='<span class="mu">尚未选择门店，点上方「选择门店（全国）」从全国门店里挑。</span>';return}c.innerHTML=qtSelected.map(id=>'<button class="chip on" data-store="'+escA(String(id))+'">'+esc(storeDisplayName(id))+' ✕</button>').join('');c.querySelectorAll('.chip').forEach(b=>b.onclick=()=>{const id=b.dataset.store;qtSelected=qtSelected.filter(x=>x!==id);renderQueueTrendStores();refreshQueueView()})}
function applyQueueStores(ids){qtSelected=(ids||[]).map(String);rememberStores('sushiro_qt_stores',qtSelected);renderQueueTrendStores();refreshQueueView()}
function applyCalendarStores(ids){selStores=(ids||[]).map(String);rStoreChoices();rC()}
let allStoresCache=null;
async function ensureAllStores(){if(allStoresCache)return allStoresCache;try{const d=await safeFetch('/api/queue/stores');allStoresCache=d.stores||[]}catch(e){allStoresCache=[]}return allStoresCache}
function storeDisplayName(id){id=String(id);const c=(allStoresCache||[]).find(s=>String(s.id)===id);if(c)return c.name||id;const a=(stores||[]).find(s=>String(s.id)===id);if(a)return a.nickname||a.name||id;const p=(qtPanels||[]).find(x=>String(x.store_id)===id);if(p)return p.store_name||id;const t=(qtTrendStores||[]).find(x=>String(x.store_id)===id);if(t)return t.store_name||id;return id}
function openStorePicker(opts){opts=opts||{};let ov=el('storePicker');if(!ov){ov=document.createElement('div');ov.id='storePicker';ov.className='ov';document.body.appendChild(ov)}ov._sel=new Set((opts.selected||[]).map(String));ov._multi=opts.multi!==false;ov._onConfirm=opts.onConfirm||function(){};ov.innerHTML='<div class="ovc"><div class="fl ai jb mb16"><b>选择门店（全国）</b><button class="bt bt-w bt-s" onclick="closeStorePicker()">关闭</button></div><input id="spSearch" type="text" placeholder="搜城市 / 门店名 / 区" oninput="renderStorePickerList()"><div id="spList" class="splist mt8"><span class="mu">加载中…</span></div><div class="fl ai jb mt16"><span class="mu" id="spCount"></span><button class="bt bt-r" onclick="confirmStorePicker()">确定</button></div></div>';ov.onclick=e=>{if(e.target===ov)closeStorePicker()};ov.classList.remove('hid');ov.style.display='flex';ensureAllStores().then(()=>renderStorePickerList())}
function closeStorePicker(){const ov=el('storePicker');if(ov){ov.classList.add('hid');ov.style.display='none'}}
function renderStorePickerList(){const ov=el('storePicker');if(!ov)return;const sel=ov._sel,q=(el('spSearch').value||'').trim().toLowerCase();const list=(allStoresCache||[]).filter(s=>{if(!q)return true;return[s.name,s.nameKana,s.area,s.address].some(v=>String(v||'').toLowerCase().includes(q))});const groups={};list.forEach(s=>{const city=s.nameKana||s.area||'其他';(groups[city]=groups[city]||[]).push(s)});const cities=Object.keys(groups).sort();el('spList').innerHTML=cities.map(city=>'<div class="spgroup"><div class="spcity">'+esc(city)+' <span class="mu">('+groups[city].length+')</span></div>'+groups[city].map(s=>{const id=String(s.id),on=sel.has(id),wait=s.wait||0,open=s.storeStatus==='OPEN',tk=String(s.netTicketStatus||'').toUpperCase(),tkOpen=tk==='ONLINE'||tk.indexOf('OPEN')>=0;const badges='<span class="spb '+(open?'ok':'mut')+'">'+(open?'营业':'休息')+'</span>'+(wait>0?'<span class="spb warn">等位'+wait+'分</span>':'')+(tkOpen?'<span class="spb ok">可取号</span>':'');return'<label class="sprow'+(on?' on':'')+'"><input type="checkbox" '+(on?'checked':'')+' onchange="toggleStorePick(\''+escA(id)+'\',this.checked)"><div class="spname">'+esc(s.name||id)+'<div class="mu">'+esc(s.area||'')+'</div></div><div class="spbs">'+badges+'</div></label>'}).join('')+'</div>').join('')||'<span class="mu">没找到匹配门店</span>';el('spCount').textContent='已选 '+sel.size+' 家'}
function toggleStorePick(id,on){const ov=el('storePicker');if(!ov)return;if(!ov._multi){ov._sel.clear();if(on)ov._sel.add(String(id));renderStorePickerList();return}if(on)ov._sel.add(String(id));else ov._sel.delete(String(id));el('spCount').textContent='已选 '+ov._sel.size+' 家'}
function confirmStorePicker(){const ov=el('storePicker');if(!ov)return;const ids=Array.from(ov._sel);closeStorePicker();(ov._onConfirm||function(){})(ids)}
function onNtModeChange(){const m=el('ntMode')?el('ntMode').value:'time',w=el('ntTimeWrap');if(w)w.classList.toggle('hid',m==='on_open')}
async function refreshQueueView(){await loadQueueLive();loadQueueRecommend();await loadNetTicketRoutine();await loadNetTicketPlan();await loadQueueAlerts();await loadQueueAlertStatus()}
// 多门店排队压力推荐：复用单店 advisor，按压力从低到高排序。
async function loadQueueRecommend(){const box=el('qtRecommend');if(!box)return;const ids=(qtSelected||[]).slice(0,6);if(ids.length<2){box.innerHTML='';return}box.innerHTML='<div class="ci">正在比较各店排队压力…</div>';try{const advs=await Promise.all(ids.map(id=>safeFetch('/api/queue/advisor?store='+encodeURIComponent(id)).catch(()=>null)));const items=advs.filter(Boolean).map(a=>{const p=a.pressure||{},c=a.current||{};return{name:a.store_name||a.store_id,level:p.level||'unknown',score:p.level==='unknown'?9999:(p.score||0),label:p.label||'数据不足',trend:p.trend_label||'',wait:c.official_wait_minutes||0,groups:c.waiting_groups||0,open:(c.store_status||'').toUpperCase()==='OPEN'}});if(!items.length){box.innerHTML='';return}items.sort((a,b)=>a.score-b.score);const best=items[0];const cards=items.map((x,i)=>'<div class="rec-card'+(i===0&&x.level!=='unknown'?' rec-best':'')+'"><div class="fl ai jb g8"><b>'+esc(x.name)+'</b><span class="answer-chip" style="padding:2px 8px"><strong class="'+pressureClass(x.level)+'">'+esc(x.label)+'</strong></span></div><div class="mu mt8">'+(x.level==='unknown'?'实时数据不足':('前面约 '+fmtN(x.groups)+' 桌 · 官方等待 '+fmtN(x.wait)+' 分'+(x.trend?' · '+esc(x.trend):'')))+'</div></div>').join('');const lead=best.level==='unknown'?'各店实时数据暂不足，先看下方实时排队。':('压力最低：<b>'+esc(best.name)+'</b>（'+esc(best.label)+'），更可能快点吃上。');box.innerHTML='<div class="cd-t" style="margin-bottom:8px">去哪家更快 <span class="tag read">只读</span></div><div class="answer-lead" style="font-size:15px">'+lead+'</div><div class="rec-grid mt8">'+cards+'</div>'}catch(e){box.innerHTML=''}}
let qtAlerts=[];
async function loadQueueAlerts(){try{const d=await safeFetch('/api/queue/alerts');qtAlerts=(d&&d.rules)||[];renderTicketReminderCard()}catch(e){}}
function alertNoList(v){return Array.from(new Set(String(v||'').split(/[，,\s]+/).map(x=>parseInt(x,10)).filter(x=>x>0)))}
async function removeQueueAlertByKey(key){try{let base=qtAlerts||[];try{const d=await safeFetch('/api/queue/alerts');base=(d&&d.rules)||base}catch(e){}const before=base.length;qtAlerts=base.filter(r=>qaRuleKey(r)!==key);if(qtAlerts.length===before){toast('没有找到这条提醒');return}await saveQueueAlerts();toast('已删除提醒')}catch(e){toast('删除提醒失败：'+String(e.message||e))}}
async function saveQueueAlerts(){try{const d=await safeFetch('/api/queue/alerts',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({rules:qtAlerts})});qtAlerts=(d&&d.rules)||qtAlerts;await loadQueueAlertStatus()}catch(e){toast('保存提醒失败')}}
async function loadQueueLive(){const box=el('qtLive');if(!box)return;box.innerHTML='<div class="skeleton skk"></div>';if(qtSelected.length){try{const ids=qtSelected.slice(0,6);const panels=await Promise.all(ids.map(id=>safeFetch('/api/queue/live?store='+encodeURIComponent(id)).catch(()=>null)));qtPanels=panels.filter(Boolean);renderQueueLivePanels(qtPanels);fillNetTicketStores()}catch(e){box.innerHTML=loadErrBoxHTML(e,'loadQueueLive()','实时排队')}return}qtPanels=[];fillNetTicketStores();const p=new URLSearchParams();p.set('limit','8');try{const d=await safeFetch('/api/queue/stores?'+p.toString());renderQueueLive(d.stores||[])}catch(e){box.innerHTML=loadErrBoxHTML(e,'loadQueueLive()','实时排队')}}
let qtPanels=[],ntPlan={},ntRoutine={};
function netTimeDisp(hhmm){hhmm=String(hhmm||'').replace(/\D/g,'').slice(0,4);while(hhmm.length<4)hhmm='0'+hhmm;return hhmm.slice(0,2)+':'+hhmm.slice(2,4)}
function fillNetTicketStores(){const ids=(qtSelected&&qtSelected.length)?qtSelected.map(String):(qdSelected&&qdSelected.length?qdSelected.map(String):qtPanels.map(p=>String(p.store_id)));const opts=ids.length?ids.map(id=>{const p=qtPanels.find(x=>String(x.store_id)===id);const nm=p?(p.store_name||id):storeDisplayName(id);return'<option value="'+escA(id)+'">'+esc(nm)+'</option>'}).join(''):'<option value="">先在上方选关注门店</option>';const sel=el('ntStore');if(sel){const prev=sel.value||(ntPlan&&ntPlan.store_id?String(ntPlan.store_id):'');sel.innerHTML=opts;if(prev&&ids.includes(prev))sel.value=prev}const rsel=el('nrStore');if(rsel){const prev=rsel.value||(ntRoutine&&ntRoutine.store_id?String(ntRoutine.store_id):'');rsel.innerHTML=opts;if(prev&&ids.includes(prev))rsel.value=prev}}
async function loadNetTicketPlan(){try{const p=await safeFetch('/api/queue/ticket/plan');ntPlan=p||{};fillNetTicketStores();if(el('ntTime')&&p.target_time)el('ntTime').value=netTimeDisp(p.target_time);if(el('ntStore')&&p.store_id)el('ntStore').value=String(p.store_id);if(el('ntMode'))el('ntMode').value=(p.trigger_mode==='on_open')?'on_open':'time';onNtModeChange();renderNetTicketStatus(p)}catch(e){}}
async function loadNetTicketRoutine(){try{const d=await safeFetch('/api/queue/ticket/routine');ntRoutine=(d&&d.routine)||{};if(d&&d.plan)ntPlan=d.plan;fillNetTicketStores();if(el('nrStore')&&ntRoutine.store_id)el('nrStore').value=String(ntRoutine.store_id);if(el('nrMeal')&&ntRoutine.target_meal_time)el('nrMeal').value=netTimeDisp(ntRoutine.target_meal_time);if(el('nrTravel'))el('nrTravel').value=ntRoutine.travel_minutes||0;if(el('nrSafety'))el('nrSafety').value=(ntRoutine.notify_before_minutes==null?(ntRoutine.safety_minutes==null?10:ntRoutine.safety_minutes):ntRoutine.notify_before_minutes);renderNetTicketRoutineStatus(ntRoutine)}catch(e){const b=el('nrStatus');if(b)b.innerHTML='<span class="mu">每日提醒状态读取失败：</span><code style="word-break:break-all">'+esc(String(e.message||e))+'</code> <button class="bt bt-w bt-s" onclick="loadNetTicketRoutine()">重试</button>'}}
function renderNetTicketRoutineStatus(r){
 const box=el('nrStatus');if(!box)return;r=r||{};
 if(!r.enabled){box.innerHTML='<span class="mu">未开启 Routine。开启后会按目标就餐时间倒推取号窗口，并提前提醒你手动取号；样本不足时不会乱提醒。</span>';return}
 const store=esc(r.store_name||storeDisplayName(r.store_id)||r.store_id||''),meal=r.target_meal_time?netTimeDisp(r.target_meal_time):'-',pickup=r.planned_pickup_time||'',pickEnd=r.planned_pickup_end_time||'',reminder=r.reminder_time||'',range=r.recommend_pickup_range?(r.recommend_pickup_range.early+'-'+r.recommend_pickup_range.late):'',window=pickup?(pickup+(pickEnd&&pickEnd!==pickup?'-'+pickEnd:'')):'',wait=r.wait_minutes_range?('预计等 '+r.wait_minutes_range.low+'-'+r.wait_minutes_range.high+' 分钟'):'等待样本不足',risk=r.risk==='high'?'风险偏高':r.risk==='medium'?'风险中等':r.risk==='low'?'风险较低':'风险待确认';
 let head='',detail='';
 switch(r.status){
  case'armed':head='已开启：今天 '+(reminder||'?')+' 提醒你取号';detail=store+' · 目标 '+meal+' 吃 · 建议取号 '+(window||range||'待确认')+' · '+wait+' · '+risk;break;
  case'needs_notify':head='已开启：需要先配置通知';detail=r.last_error||'Routine 只是提醒，不配置通知渠道就无法按时提醒你取号。';break;
  case'waiting_data':head='已开启：等待历史样本';detail=r.last_error||'这家店样本不足，暂不提醒。去“预测准确度”开启本机采集后会自动补齐。';break;
  case'missed':head='今天已错过提醒窗口';detail=r.last_error||'Routine 明天会重新规划提醒时间。';break;
  case'notified':head='今天已提醒取号';detail=store+' · 建议取号 '+(window||range||'待确认')+' · 目标 '+meal+' 吃。';break;
  case'done':head='今天已经取到号';detail=r.last_error||'如果你已经手动取到号，可以到“我有号码”继续做叫号预测。';break;
  case'error':head='Routine 保存失败';detail=r.last_error||'未知错误';break;
  default:head='已开启：等待下一次规划';detail='目标 '+meal+' 吃，后台会按历史等待倒推提醒时间。'
 }
 const notifyBtn=r.status==='needs_notify'?'<button class="bt bt-r bt-s" onclick="focusNotifySettings()">配置通知</button>':'';
 box.innerHTML='<b>'+esc(head)+'</b><div class="mu mt8">'+esc(detail)+(r.basis?'<br>'+esc(r.basis):'')+'</div><div class="fl g8 fw mt8">'+notifyBtn+'<button class="bt bt-w bt-s" onclick="openSettingsFold(&quot;fold-sm&quot;)">提升预测准确度</button><button class="bt bt-w bt-s" onclick="refreshNetTicketRoutineNow()">重新试算今天</button></div>'
}
function renderNetTicketStatus(p){
 const box=el('ntStatus');if(!box)return;p=p||{};
 const store=esc(p.store_name||p.store_id||''),tt=p.target_time?netTimeDisp(p.target_time):'';
 if(!p.enabled){box.innerHTML=!hc?'<div class="notice">自动取号需要寿司郎通行证。现在还没配置——点下方「拿通行证」获取后，才能定时或一开放就自动远程取号。</div><div class="fl g8 fw mt8"><button class="bt bt-r bt-s" onclick="startAuth()">拿通行证</button></div>':'<span class="mu">选门店和时间，点「启用」即可设置自动取号计划；这不是只读功能，启用前会再次确认。</span>';return}
 switch(p.status){
  case 'success':box.innerHTML='<b>已自动取号 '+esc(p.number||'(详见我的单据)')+'</b><div class="mu mt8">'+store+' · 电脑已停止当天取号轮询；现在用手机寿司郎小程序查看排队信息更稳。</div>';break;
  case 'issued_unknown':box.innerHTML='<b>⚠️ 官方提示已经发过号，但本地号码未知</b><div class="mu mt8">'+store+' '+tt+'：'+esc(p.last_error||'不要重复取号，请用手机寿司郎小程序查看排队号。')+'<br>电脑已停止当天取号轮询，避免影响手机端查看。</div>';break;
  case 'retrying':box.innerHTML='<b>⏳ 取号暂未成功，窗口内继续重试</b><div class="mu mt8">'+store+' '+tt+'：'+esc(p.last_error||'如果提示凭证需要刷新，请先重新认证')+'</div>';break;
  case 'error':{const authErr=/E010|error\\.server|凭证|认证/.test(String(p.last_error||''));box.innerHTML='<b>⚠️ 取号失败</b><div class="mu mt8">'+store+' '+tt+'：'+esc(p.last_error||'未知错误')+'<br>'+(authErr?'寿司郎凭证会过期或被手机端登录顶掉，请先重置认证。':'改时间后重新启用可重试。')+'</div>'+(authErr?'<div class="mt8"><button class="bt bt-r bt-s" onclick="resetAuthAndStart()">重置并重新认证</button></div>':'');break;}
  case 'expired':box.innerHTML='<b>⏰ 未在窗口内取到号</b><div class="mu mt8">'+store+' '+tt+'：超时已放弃，可重新启用。</div>';break;
  default:box.innerHTML='<b>⏳ 已设定：'+tt+' 自动取号</b><div class="mu mt8">'+store+' · 到点(约 '+tt+')自动远程取号并发一次通知。取到后电脑会停止当天轮询。</div>';
 }
}
async function refreshNetTicketRoutineNow(){if(!ntRoutine||!ntRoutine.enabled){await loadNetTicketRoutine();return}const before=ntRoutine.notify_before_minutes==null?(ntRoutine.safety_minutes==null?10:ntRoutine.safety_minutes):ntRoutine.notify_before_minutes;try{const d=await safeFetch('/api/queue/ticket/routine',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled:true,store:ntRoutine.store_id,store_name:ntRoutine.store_name||storeDisplayName(ntRoutine.store_id),target_meal_time:ntRoutine.target_meal_time,travel_minutes:ntRoutine.travel_minutes||0,notify_before_minutes:before})});if(d.error){toast(d.error);return}ntRoutine=d.routine||{};if(d.plan)ntPlan=d.plan;fillNetTicketStores();renderNetTicketRoutineStatus(ntRoutine);renderNetTicketStatus(ntPlan);toast('已重新试算今天')}catch(e){toast('重新试算失败')}}
async function saveNetTicketRoutine(enabled){const sel=el('nrStore'),store=sel?sel.value:'',meal=el('nrMeal')?.value||'',travel=Math.max(0,parseInt(el('nrTravel')?.value||'0',10)||0),before=Math.max(0,parseInt(el('nrSafety')?.value||'0',10)||0);if(enabled){if(!store){toast('请先选门店');return}if(!meal){toast('请填想几点吃');return}if(!nfc){toast('启用 Routine 前必须先配置通知渠道');focusNotifySettings();return}if(!await confirmDialog('启用每日取号提醒 Routine？\\n每天会按目标就餐时间倒推取号窗口，并提前提醒你手动取号。\\n不会自动向寿司郎提交取号请求。'))return}else if(ntRoutine&&ntRoutine.enabled){if(!await confirmDialog('关闭每日取号提醒 Routine？\\n这只会停止未来提醒，不会取消已经拿到的排队号。'))return}const sn=(sel&&sel.selectedOptions[0])?sel.selectedOptions[0].textContent:'';try{const d=await safeFetch('/api/queue/ticket/routine',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled:enabled,store:store,store_name:sn,target_meal_time:compactTime(meal),travel_minutes:travel,notify_before_minutes:before})});if(d.error){toast(d.error);return}ntRoutine=d.routine||{};if(d.plan)ntPlan=d.plan;fillNetTicketStores();renderNetTicketRoutineStatus(ntRoutine);renderNetTicketStatus(ntPlan);toast(enabled?'已开启取号提醒 Routine':'已关闭 Routine')}catch(e){toast('保存 Routine 失败')}}
async function saveNetTicketPlan(enabled){const sel=el('ntStore'),tEl=el('ntTime'),modeEl=el('ntMode'),store=sel?sel.value:'',mode=modeEl?modeEl.value:'time',t=tEl?tEl.value:'';if(enabled){if(!store){toast('请先选门店');return}if(mode==='time'&&!t){toast('请填取号时间');return}const tip=mode==='on_open'?'门店一开放线上取号就会自动远程取号。':'到 '+t+' 会自动远程取号。';if(!await confirmDialog('启用自动取号计划？\\n'+tip+'\\n取到号后请尽快到店；这不是只读功能。'))return}else if(ntPlan&&ntPlan.enabled){if(!await confirmDialog('取消自动取号计划？\\n这只会停止本工具未来自动取号，不会取消已经拿到的排队号。'))return}const sn=(sel&&sel.selectedOptions[0])?sel.selectedOptions[0].textContent:'',tt=t?t.replace(':',''):'';try{const p=await safeFetch('/api/queue/ticket/plan',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled:enabled,store:store,store_name:sn,trigger_mode:mode,target_time:tt})});if(p.error){toast(p.error);return}ntPlan=p;renderNetTicketStatus(p);toast(enabled?(mode==='on_open'?'已启用：门店一开放线上取号就自动取号':('已启用定时取号：'+netTimeDisp(tt)+' 自动取号')):'已取消取号计划')}catch(e){toast('保存失败')}}
async function recoverNetTicketStatus(){try{const d=await safeFetch('/api/queue/ticket/status',null,12000);const t=d.ticket||{},p=d.plan||{};ntPlan=p;renderNetTicketStatus(p);lR();toast('已恢复当前排队号：'+(t.number||p.number||'(详见我的单据)'))}catch(e){toast('恢复失败：'+String(e.message||e))}}
async function cancelNetTicket(){if(!await confirmDialog('危险操作：取消当前排队号？\\n这会取消寿司郎小程序里的排队号，取消后不可恢复。\\n如果你只是想停止本工具，请点“取消计划”或“停止”。'))return;try{const d=await safeFetch('/api/queue/ticket/cancel',{method:'POST'});if(d.error){toast('取消失败：'+d.error);return}toast('已取消排队号');await loadNetTicketPlan();loadActiveTickets(true);if(typeof lR==='function')lR()}catch(e){toast('取消失败：'+String(e.message||e))}}
function sparkSVG(arr){if(!arr||arr.length<2)return'';const w=140,h=34,mn=Math.min(...arr),mx=Math.max(...arr),rg=(mx-mn)||1,n=arr.length,dx=w/(n-1);const pts=arr.map((v,i)=>(i*dx).toFixed(1)+','+(h-3-((v-mn)/rg)*(h-6)).toFixed(1)).join(' ');return'<svg class="spark" viewBox="0 0 '+w+' '+h+'" preserveAspectRatio="none"><polyline points="'+pts+'"/></svg>'}
function waitLevel(s){const eta=(s.eta_minutes!=null)?s.eta_minutes:(s.server_wait_minutes||0),cap=s.wait_time_cap||180,pct=eta<=0?0:Math.max(5,Math.min(100,Math.round(eta/cap*100))),lvl=eta<=0?'g':eta<=30?'g':eta<=90?'y':'r';return{eta:eta,pct:pct,lvl:lvl}}
function renderQueueLivePanels(rows){const box=el('qtLive');if(!box)return;if(!rows.length){box.innerHTML='<div class="empty">还没拿到实时排队数据，请刷新或换一家门店。<div class="mt8"><button class="bt bt-w bt-s" onclick="refreshQueueView()">重试</button></div></div>';return}const note=currentUIMode()==='advanced'?'门店、叫号、在等桌数为公开实时信息；远程取号是会执行操作的实验性功能，确认后才会提交。':'门店、叫号、在等桌数为公开实时信息；简化版保持只读，不会替你取号。';box.innerHTML='<div class="queue-live-grid">'+rows.map(s=>{const open=s.online_open||s.store_status==='OPEN',card=open?'open':'closed',status=open?'可取号':'暂停',etaTxt=(s.eta_minutes!=null)?(s.eta_minutes+' 分钟'):(s.server_wait_minutes?(s.server_wait_minutes+' 分钟*'):'—'),called15=s.called_15m!=null?('+'+s.called_15m):'待收集',rate=s.rate_per_min!=null?(s.rate_per_min.toFixed(1)+' 桌/分'):'待收集',wl=waitLevel(s),trend=(s.called_15m>0)?'↑':'';return'<article class="queue-live-card '+card+'"><div class="queue-live-top"><div class="queue-live-name"><b>'+esc(s.store_name||s.store_id)+'</b><span>'+esc([s.store_status||'-',s.net_ticket_status||'-'].join(' · '))+'</span></div><span class="queue-status '+(open?'ok':'bad')+'">'+esc(status)+'</span></div><div class="queue-live-main"><div class="queue-call"><span>当前叫号</span><strong>'+esc(s.called_no||'—')+' <em>'+esc(trend)+'</em></strong></div><div class="queue-spark">'+(sparkSVG(s.spark)||'<span class="mu">小折线待收集</span>')+'</div></div><div class="queue-metrics"><div class="queue-metric"><span>前面</span><b>'+fmtN(s.wait_groups||0)+' 桌</b></div><div class="queue-metric"><span>约等待</span><b>'+esc(etaTxt)+'</b></div><div class="queue-metric"><span>近15分钟</span><b>'+esc(called15)+'</b></div></div><div class="queue-meter" title="拥挤度"><i class="lv-'+wl.lvl+'" style="width:'+wl.pct+'%"></i></div><div class="queue-live-foot"><span>均速 '+esc(rate)+' · 拥挤度 '+wl.pct+'%'+(s.tables_capacity?(' · 桌位 '+s.tables_capacity+(s.counters_capacity?(' / 吧台 '+s.counters_capacity):'')):'')+'</span><button class="bt bt-o bt-s advanced-only" onclick="takeTicket(\''+escA(String(s.store_id||''))+'\')">远程取号</button></div></article>'}).join('')+'</div><p class="queue-live-note">'+esc(note)+'</p>'}
function renderQueueLive(rows){const box=el('qtLive');if(!box)return;if(!rows.length){box.innerHTML='<div class="empty">还没拿到门店排队数据。点上方「选择门店（全国）」搜索城市或门店名，手动选择关注门店。</div>';return}box.innerHTML='<div class="sg">'+rows.map(s=>{const wait=(s.wait==null?0:s.wait),groups=(s.groupQueuesCount==null?0:s.groupQueuesCount),status=s.storeStatus||'-',ticket=s.netTicketStatus||'-',cls=status==='OPEN'?'av':'full';return'<div class="sl '+cls+'"><div class="tm">预计 '+wait+' 分钟</div><div class="ss">'+esc(s.name||s.id)+' · '+esc(s.nameKana||s.area||'')+'</div><div class="mu mt8">在等 '+groups+' 桌 · '+esc(status)+' · '+esc(ticket)+(s.waitTimeCap?'<br>预估上限 '+esc(s.waitTimeCap)+' 分钟':'')+'</div></div>'}).join('')+'</div><p class="mu mt8">选中上方关注门店即可查看实时叫号、近15分钟叫号与均速。</p>'}
function queueStatusText(q){if(!q)return'未知';if(q.needs_auth)return'凭证需更新';if(q.needs_background)return'需开启';if(q.needs_data_refresh)return'需更新';return'正常'}
function queueTypeName(t){return t==='weekday'?'工作日':t==='workday'?'调休工作日':t==='weekend'?'周末':t==='holiday'?'节假日':t}
function confText(v){return v==='high'?'高':v==='medium'?'中':v==='low'?'低':'无'}

async function lSm(){await ensureStores();await loadSampling()}
async function loadSampling(){try{const d=await(await fetch('/api/sampling')).json();spCfg=d.config||{};spState=d.state||{};spAutoStart=d.autostart||{};spQueueState=d.queue_state||{};fillSamplingForm();renderSamplingStores();renderSamplingState();renderDashboardSamplingCard()}catch(e){const ss=el('sampleState');if(ss)ss.innerHTML='<div class="ci bad">预测准确度状态加载失败</div>';renderDashboardSamplingCard()}}
function fillSamplingForm(){const set=(id,fn)=>{const e=el(id);if(e)fn(e)};set('spEnabled',e=>e.checked=!!spCfg.enabled);set('spAuto',e=>e.checked=!!spCfg.auto_start);set('spInterval',e=>e.value=spCfg.interval_seconds||300);set('spStart',e=>e.value=timeInputValue(spCfg.active_start||'100000'));set('spEnd',e=>e.value=timeInputValue(spCfg.active_end||'220000'))}
function renderSamplingStores(){const c=el('samplingStores'),h=el('sampleStoreHint');if(!c)return;if(!stores.length){c.innerHTML='<span class="mu">本机采集需要寿司郎认证；只看实时排队不用。</span>';if(h)h.textContent='先获取凭证后，才能记录你常去门店的历史变化。';return}const chosen=(spCfg.store_ids||[]).map(String);c.innerHTML=stores.map(s=>'<button class="chip '+(chosen.includes(String(s.id))?'on':'')+'" data-store="'+escA(String(s.id))+'">'+esc(s.nickname||s.name||s.id)+'</button>').join('');c.querySelectorAll('.chip').forEach(b=>b.onclick=()=>{b.classList.toggle('on');renderSamplingStoreHint()});renderSamplingStoreHint()}
function renderSamplingStoreHint(){const h=el('sampleStoreHint');if(!h)return;const chosen=Array.from(document.querySelectorAll('#samplingStores .chip.on')).map(x=>x.dataset.store);if(chosen.length){h.textContent='当前记录 '+chosen.length+' 家指定门店。';return}const pref=(pr.selected_stores||[]).map(storeName).filter(Boolean);h.textContent=pref.length?'当前跟随预约/取号门店：'+pref.join('、'):'当前跟随凭证里保存的门店。'}
function samplingPayload(){const ids=Array.from(document.querySelectorAll('#samplingStores .chip.on')).map(x=>x.dataset.store);return{enabled:el('spEnabled').checked,auto_start:el('spAuto').checked,interval_seconds:+el('spInterval').value||300,active_start:compactTime(el('spStart').value||'10:00'),active_end:compactTime(el('spEnd').value||'22:00'),store_ids:ids,use_preference_stores:ids.length===0}}
function renderSamplingState(){const s=spState||{},a=spAutoStart||{},q=spQueueState||{},next=s.next_run_at?new Date(s.next_run_at).toLocaleString():'-',last=s.last_run_at?new Date(s.last_run_at).toLocaleString():'-',msg=s.last_error||s.message||q.message||'无',bad=(s.last_error||q.needs_auth)&&!/跳过|时间窗|暂无|正在运行/.test(s.last_error||'');el('sampleState').innerHTML=chip('状态',s.running?'运行中':(s.enabled?'已启用':'未启动'),s.running?'ok':s.enabled?'warn':'')+chip('开机自启动',a.enabled?'已配置':a.supported?'未配置':'不支持',a.enabled?'ok':'warn')+chip('下次',next,'ok')+chip('上次',last,'ok')+chip('样本',s.snapshots||0,'ok')+chip('门店失败',s.store_errors||0,(s.store_errors||0)?'warn':'ok')+chip('凭证',q.auth_ok?'可用':'需更新',q.auth_ok?'ok':'bad')+chip('最近结果',msg,bad?'bad':'ok');renderSettingsStatus()}
async function saveSampling(quiet){spCfg=samplingPayload();try{const d=await(await fetch('/api/sampling',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(spCfg)})).json();if(d.error){if(!quiet)toast(d.error);return false}spCfg=d.config||spCfg;spState=d.state||spState;renderSamplingStores();renderSamplingState();renderDashboardSamplingCard();if(!quiet)toast(spState.running?'预测配置已保存，后台已按新配置重启':'预测配置已保存');return true}catch(e){if(!quiet)toast('保存失败');return false}}
async function startSampling(){if(el('spEnabled'))el('spEnabled').checked=true;if(!await saveSampling(true))return;try{const d=await(await fetch('/api/sampling/start',{method:'POST'})).json();if(d.error){toast(d.error);return}spState=d.state||spState;await loadSampling();toast('已开启本机持续采集')}catch(e){toast('启动失败')}}
async function stopSampling(){try{const d=await(await fetch('/api/sampling/stop',{method:'POST'})).json();spState=d.state||spState;renderSamplingState();renderDashboardSamplingCard()}catch(e){toast('停止失败')}}
async function runSampleOnce(){if(!await saveSampling(true))return;const box=el('sampleResult');box.classList.remove('hid');box.textContent='收集中';try{const d=await(await fetch('/api/sampling/once',{method:'POST'})).json();spState=d.state||spState;renderSamplingState();renderDashboardSamplingCard();const r=d.result||{};box.innerHTML=r.skipped?'本轮跳过：'+esc(r.skip_reason):'<b>收集完成</b><br>'+esc((r.stores||[]).map(x=>{const parts=[];parts.push(x.error||((x.slots||0)+' 条时段'));if(x.queue_observed)parts.push('排队 '+(x.queue_wait_groups||0)+' 组');else if(x.queue_error)parts.push('排队失败');return(x.store_name||x.store_id)+': '+parts.join('，')}).join('\\n')).replaceAll('\\n','<br>');if(cp==='qd')toast(r.skipped?'本轮跳过：'+(r.skip_reason||'未知原因'):'收集完成：'+(r.queue_snapshots||0)+' 条排队快照，'+(r.snapshots||0)+' 条时段')}catch(e){box.innerHTML='收集失败';renderDashboardSamplingCard()}}
function usePrefSamplingStores(){document.querySelectorAll('#samplingStores .chip').forEach(x=>x.classList.remove('on'));renderSamplingStoreHint()}
async function setBootSampling(enabled){try{const d=await(await fetch('/api/sampling/autostart',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled})})).json();if(d.error){toast(d.error);return}spAutoStart=d.autostart||{};if(cp==='se'||cp==='qd')await loadSampling();toast(enabled?'已配置开机自启动':'已取消开机自启动')}catch(e){toast('操作失败')}}

let pendingSnTarget=null;
function snFromSlot(store_id,date,start,end){pendingSnTarget={store_id:String(store_id),date:String(date),start_after:String(start),start_before:String(end||start)};go('sn')}
async function bookSlotDirect(store_id,date,start,end,store_name){const when=fT(start)+(end?'-'+fT(end):'');if(!await confirmDialog({title:'直接预约这个时段',body:'会向寿司郎提交预约：\\n'+(store_name||store_id)+'\\n'+fD(date)+' '+when+'\\n这是会执行操作，不是只读查看；成功后可在「我的单据」查看。',ok:'确认预约',cancel:'再想想'}))return;await submitGuard('book:'+store_id+':'+date+':'+start,async()=>{try{const d=await safeFetch('/api/engine/booking',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({store:String(store_id),date:String(date),start:String(start),end:String(end||'')})});if(d.error){toast('预约失败：'+d.error);return}toast('已开始预约这个时段，进度看首页或我的单据');await loadStatus();go('da')}catch(e){toast('预约失败：'+String(e.message||e))}})}
async function lSn(){await ensureStores();if(!el('snRows').children.length)addSn();await loadSnPlan();if(pendingSnTarget){const t=pendingSnTarget;pendingSnTarget=null;const rows=el('snRows');if(rows.children.length===1&&!rows.querySelector('input').value)rows.innerHTML='';addSn(t);rows.lastElementChild?.scrollIntoView({block:'center'})}else if(!((pr.selected_stores||[]).length)){/* 新人进 sn 页没选门店偏好：自动展开偏好区并定位，避免面对空目标行不知从何下手 */setTimeout(expandSnPrefs,80)}}
async function ensureStores(){if(stores.length)return;try{stores=await(await fetch('/api/stores')).json();selStores=stores.map(s=>String(s.id));}catch(e){}}
function storeOpts(v){return stores.map(s=>'<option value="'+escA(String(s.id))+'" '+(String(s.id)===String(v)?'selected':'')+'>'+esc(s.nickname||s.name||s.id)+'</option>').join('')}
function dateInputValue(v){v=String(v||'');return /^\d{8}$/.test(v)?v.slice(0,4)+'-'+v.slice(4,6)+'-'+v.slice(6,8):v}
function timeInputValue(v){v=String(v||'');return /^\d{6}$/.test(v)?v.slice(0,2)+':'+v.slice(2,4):/^\d{4}$/.test(v)?v.slice(0,2)+':'+v.slice(2,4):v}
function compactDate(v){v=String(v||'').trim();return /^\d{4}-\d{2}-\d{2}$/.test(v)?v.replaceAll('-',''):v}
function compactTime(v){v=String(v||'').trim();return /^\d{2}:\d{2}$/.test(v)?v.replace(':',''):v.replaceAll(':','')}
function validDate8(v){if(!/^\d{8}$/.test(v))return false;const d=new Date(v.slice(0,4)+'-'+v.slice(4,6)+'-'+v.slice(6,8));return !Number.isNaN(d.getTime())&&d.toISOString().slice(0,10).replaceAll('-','')===v}
function timeSec(v){v=compactTime(v);if(!/^(?:\d{4}|\d{6})$/.test(v))return -1;const h=+v.slice(0,2),m=+v.slice(2,4),s=v.length===6?+v.slice(4,6):0;return h>=0&&h<=23&&m>=0&&m<=59&&s>=0&&s<=59?h*3600+m*60+s:-1}
function addSnErr(row,msg){const d=document.createElement('div');d.className='inline-err';d.textContent=msg;row.appendChild(d)}
function addSn(t={}){const c=el('snRows'),d=document.createElement('div');d.className='sn-row';d.innerHTML='<div class="fg"><label>日期</label><input type="date" value="'+escA(dateInputValue(t.date||''))+'"></div><div class="fg"><label>最早</label><input type="time" value="'+escA(timeInputValue(t.start_after||'1930'))+'"></div><div class="fg"><label>最晚</label><input type="time" value="'+escA(timeInputValue(t.start_before||'2030'))+'"></div><div class="fg"><label>门店</label><select>'+storeOpts(t.store_id||stores[0]?.id||'')+'</select></div><button class="bt bt-o bt-s" onclick="this.parentElement.remove()">删除</button>';c.appendChild(d)}
function readSnTargets(){const rows=Array.from(el('snRows').children);let ok=true;const targets=[];rows.forEach(r=>{r.querySelector('.inline-err')?.remove();const i=r.querySelectorAll('input'),s=r.querySelector('select'),date=compactDate(i[0].value),start=compactTime(i[1].value),end=compactTime(i[2].value),ss=timeSec(start),es=timeSec(end);if(!date&&!start&&!end&&!s.value)return;if(!validDate8(date)){ok=false;addSnErr(r,'日期无效');return}if(ss<0||es<0){ok=false;addSnErr(r,'时间无效');return}if(ss>=es){ok=false;addSnErr(r,'最晚时间必须晚于最早时间');return}if(!s.value){ok=false;addSnErr(r,'请选择门店');return}targets.push({date,start_after:start,start_before:end,store_id:s.value})});return{ok,targets}}
function snTargets(){const r=readSnTargets();return r.ok?r.targets:[]}
async function saveSn(){const read=readSnTargets();if(!read.ok)return;if(!read.targets.length){toast('请至少添加一个有效目标时段');return}try{const d=await(await fetch('/api/sniper/plan',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({targets:read.targets})})).json();if(d.error){toast(d.error);return}renderSnPlan(d.plan);toast('已保存蹲未来预约计划')}catch(e){toast('保存失败')}}
async function loadSnPlan(){try{const d=await(await fetch('/api/sniper/plan')).json();if(d.targets?.length){el('snRows').innerHTML='';d.targets.forEach(addSn)}renderSnPlan(d)}catch(e){}}
function renderSnPlan(p){const c=el('snPlan'),ts=p?.targets||[];if(!ts.length){c.innerHTML='<div class="empty">还没有蹲未来预约目标。点“添加目标时段”，填日期、门店和时间窗。</div>';return}c.innerHTML='<table class="tbl"><thead><tr><th>目标时段</th><th>开放窗口</th><th>状态</th><th>尝试</th><th>最后错误</th></tr></thead><tbody>'+ts.map(t=>'<tr><td>'+esc(t.store_id)+'<br>'+esc(t.date)+' '+esc(fT(t.start_after))+'-'+esc(fT(t.start_before))+'</td><td>'+esc(t.open_at?new Date(t.open_at).toLocaleString():'-')+'<br>'+(t.countdown_seconds>0?Math.ceil(t.countdown_seconds/60)+' 分钟后':'窗口内/已结束')+'</td><td>'+esc(t.status||'-')+'</td><td>'+esc(t.attempts||0)+'</td><td>'+esc(t.last_error||'')+'</td></tr>').join('')+'</tbody></table>'}
async function startSn(){const read=readSnTargets();if(!read.ok)return;if(!read.targets.length){toast('请至少添加一个有效目标时段');return}if(!await ensureNotifyConfigured('抢到未来预约'))return;if(!await confirmDialog('启动蹲未来预约时段？\\n到开放窗口会自动尝试创建未来预约；抢到后会停止。\\n不会取消已有预约或排队号。'))return;await submitGuard('startSn',async()=>{try{const d=await(await fetch('/api/sniper/start',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({targets:read.targets})})).json();if(d.error){toast(d.error);return}await loadStatus();await loadSnPlan();toast('蹲未来预约计划已启动，抢到的预约会出现在“我的单据”')}catch(e){toast('启动失败')}})}

async function lR(){const c=el('rc');if(!hc){c.innerHTML='<div class="empty"><div class="mascot-wrap">'+mascotSVG('plain',56)+'</div>查看官方预约和排队号需要先拿一次通行证 🎫（约 3 分钟）。<div class="mt8"><button class="bt bt-r bt-s" onclick="startAuth()">去拿通行证</button><button class="bt bt-w bt-s" onclick="go(\'qt\')">先看排队</button></div></div>';return}c.innerHTML='<div class="empty">正在读取你的预约和排队号。</div>';try{const d=await safeFetch('/api/reservations');if(d.error){loadStatus();c.innerHTML=loadErrBoxHTML(d.error,'lR()','我的单据');return}const items=Array.isArray(d)?d:(d.items||[]);if(!items.length){c.innerHTML='<div class="empty"><div class="mascot-wrap">'+mascotSVG('sleep',56)+'</div>当前没有预约或排队号。<div class="mt8"><button class="bt bt-r bt-s" onclick="go(\'ca\')">约未来</button><button class="bt bt-w bt-s" onclick="go(\'qt\')">看排队</button></div></div>';return}c.innerHTML='<div class="sg">'+items.map(r=>{const when=r.slot_label||[r.queueDate,fT(r.start),r.end?'-'+fT(r.end):''].filter(Boolean).join(' '),store=r.store_name||r.monitored_store_id||r.storeId||'',kind=recordKind(r);const extra=[];if(kind==='net_ticket'&&r.wait>0)extra.push('前面 '+r.wait+' 桌');if(kind==='net_ticket')extra.push(r.checkedIn?'已签到':'未签到');if(kind==='reservation')extra.push('预约时间优先');extra.push(kind==='net_ticket'?'排队号':kind==='reservation'?'预约':'类型待确认');const cancel=cancelActionHTML(r,kind);return'<div class="sl av"><div class="tm">'+esc(r.number||'-')+'</div><div class="ss">'+esc(recordStatusText(r,kind))+(store?' · '+esc(store):'')+'</div><div class="mu mt8">'+esc(when||'时间待确认')+'<br>'+esc(extra.join(' · '))+'<br>#'+esc(r.ticketId||'')+'</div>'+cancel+'</div>'}).join('')+'</div>'}catch(e){loadStatus();c.innerHTML=loadErrBoxHTML(e,'lR()','我的单据')}}
function hasReservationSchedule(r){return!!(r.slot_label||r.start||r.end)}
function recordKind(r){const k=String(r.kind||'').toLowerCase();if(k==='reservation'||k==='reservation_ticket')return'reservation';if(hasReservationSchedule(r))return'reservation';if(k==='net_ticket'||k==='netticket')return'net_ticket';if(r.wait>0||String(r.status||'').toUpperCase()==='WAITING')return'net_ticket';return'unknown'}
function recordStatusText(r,kind){const s=String(r.status||'').trim(),u=s.toUpperCase();if(kind==='reservation'){if(u==='WAITING')return'预约待到店';if(u==='RESERVED')return'已确认预约';if(u==='CHECKED_IN')return'已签到预约';return s||'已确认预约'}if(kind==='net_ticket'){if(u==='WAITING')return'排队中';if(u==='CALLED')return'已叫号';return s||'排队号'}return s||'-'}
function cancelActionHTML(r,kind){if(kind==='net_ticket')return'<div class="mt8"><button class="bt bt-o bt-s" onclick="cancelNetTicket()">取消排队号</button></div>';if(kind==='reservation'&&r.ticketId)return'<div class="mt8"><button class="bt bt-o bt-s" onclick="cancelTicket('+r.ticketId+',&quot;reservation&quot;)">取消预约</button></div>';return'<div class="mu mt8">为避免误取消，类型未确认的记录不提供取消按钮。</div>'}
async function cancelTicket(id,kind){if(kind!=='reservation'){toast('安全保护：排队号请使用“取消排队号”，不会走预约取消接口。');return}if(!await confirmDialog('危险操作：取消当前预约？\\n这会取消寿司郎小程序里的预约单，取消后不可恢复。\\n如果你只是想刷新状态，请不要点确认。'))return;try{const d=await(await fetch('/api/reservations/cancel',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({ticket_id:id,kind:'reservation'})})).json();if(d.error){toast('取消失败：'+d.error);return}toast('已取消预约');lR()}catch(e){toast('取消失败')}}
async function takeTicket(id){if(!await confirmDialog('现在远程取号？\\n这会向寿司郎提交取号请求，不是只读查看。\\n取号后请尽快到店，过号会作废。'))return;await submitGuard('takeTicket',async()=>{try{const d=await(await fetch('/api/queue/ticket',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({store:String(id)})})).json();if(d.error){toast('取号失败：'+d.error);return}const t=d.ticket||{};toast('已取号 '+(t.number||'(详见我的单据)')+'，请到“我的单据”查看');loadActiveTickets(true)}catch(e){toast('取号失败')}})}

async function ensurePrefsLoaded(){if(prefsLoaded&&Object.keys(pr||{}).length)return pr;if(!prefsLoading){prefsLoading=(async()=>{const d=await(await fetch('/api/preferences')).json();pr=d;prefsLoaded=true;return d})().finally(()=>{prefsLoading=null})}return prefsLoading}
async function lP(){try{const modeSeq=uiModeSeq;await ensurePrefsLoaded();if(modeSeq===uiModeSeq)cacheUIMode(pr.ui_mode==='advanced'?'advanced':'simple');fF(pr);dP(pr);renderBookingStores();uD();applyUIMode()}catch(e){applyUIMode()}}
function fF(p){el('pa').value=p.adult||2;el('pc').value=p.child||0;el('pt').value=p.table_type||'T';if(el('pphone'))el('pphone').value=p.phone_number||'';el('ppm').value=p.day_priority_mode||'date';el('pst').value=p.slot_strategy||'earliest';el('ptm').value=p.target_time||'1930';rT('wd',p.weekday_slots||[]);rT('sa',p.saturday_slots||[]);rT('su',p.sunday_slots||[])}
function rangeText(rs){return !rs||!rs.length?'不预约':rs.map(r=>fT(String(r.start||''))+'-'+fT(String(r.end||''))).join('、')}
function priText(v){return v==='weekend_first'?'周末优先':v==='weekday_first'?'工作日优先':'按日期优先'}
function stratText(v,t){return v==='latest'?'最晚可约':v==='closest'?'接近 '+fT(t||'1930'):'最早可约'}
function dP(p){const people=(p.adult||2)+' 成人'+((p.child||0)>0?' · '+p.child+' 儿童':'');const table=(p.table_type||'T')==='C'?'吧台':'桌位',pri=priText(p.day_priority_mode),str=stratText(p.slot_strategy,p.target_time);const notifyHint=(hc&&!nfc)?'<span class="line" style="color:#b81c22">⚠ 未配置通知，抢到预约 / 叫号提醒不会推送 —— <a href="#" onclick="focusNotifySettings();return false" style="color:#b81c22;text-decoration:underline">去设置</a></span>':'';el('ps').innerHTML='<b>'+esc(people)+'</b> · '+esc(table)+'<span class="line">优先级：'+esc(pri)+' · '+esc(str)+'</span><span class="line">工作日：'+esc(rangeText(p.weekday_slots))+'</span><span class="line">周六：'+esc(rangeText(p.saturday_slots))+'</span><span class="line">周日：'+esc(rangeText(p.sunday_slots))+'</span>'+notifyHint}
function storeName(id){const s=stores.find(x=>String(x.id)===String(id));return s?(s.nickname||s.name||s.id):id}
function orderedStoreIDs(){const all=stores.map(s=>String(s.id)),sel=(pr.selected_stores||[]).map(String).filter(id=>all.includes(id)),base=(pr.store_priority||[]).map(String).filter(id=>all.includes(id));let order=[];base.forEach(id=>{if(!order.includes(id))order.push(id)});sel.forEach(id=>{if(!order.includes(id))order.push(id)});all.forEach(id=>{if(!order.includes(id))order.push(id)});return{all,selected:sel.length?sel:all,order}}
async function searchStores(){const q=(el('storeSearch')?.value||'').trim(),box=el('storeSearchResults');if(!box)return;if(!q){box.innerHTML='<span class="mu">输入城市或门店名再搜。</span>';return}box.innerHTML='<span class="mu">搜索中…</span>';try{const d=await safeFetch('/api/queue/stores?limit=24&q='+encodeURIComponent(q));const list=d.stores||[];if(!list.length){box.innerHTML='<span class="mu">没找到匹配门店，换个关键词试试。</span>';return}const have=new Set(stores.map(s=>String(s.id)));box.innerHTML='<div class="store-result-grid">'+list.map(s=>{const id=String(s.id),added=have.has(id),nm=String(s.name||id);return'<div class="sl av"><div class="ss"><b>'+esc(nm)+'</b></div><div class="mu mt8">'+esc([s.nameKana,s.area].filter(Boolean).join(' · ')||'门店 '+id)+'</div><div class="mt8">'+(added?'<button class="bt bt-w bt-s" disabled>已添加</button>':'<button class="bt bt-r bt-s" onclick="addStoreFromSearch(\''+escA(id)+'\',\''+escA(nm)+'\')">添加</button>')+'</div></div>'}).join('')+'</div>'}catch(e){box.innerHTML='<div class="ci bad">搜索失败</div>'}}
async function addStoreFromSearch(id,name){id=String(id);if(!stores.some(s=>String(s.id)===id))stores.push({id:id,name:name,nickname:name});pr.selected_stores=(pr.selected_stores||[]).map(String);if(!pr.selected_stores.includes(id))pr.selected_stores.push(id);pr.store_priority=(pr.store_priority||[]).map(String);if(!pr.store_priority.includes(id))pr.store_priority.push(id);renderBookingStores();if(el('storeChoices'))rStoreChoices();await savePrefsPayload(prefsPayload(),true);searchStores()}
function renderBookingStores(){const box=el('bookingStores');if(!box)return;if(!stores.length){box.innerHTML='<span class="mu">拿到通行证后可在此选择门店</span>';return}const data=orderedStoreIDs(),set=new Set(data.selected);box.innerHTML=data.order.map(id=>'<div class="store-row" data-store="'+escA(id)+'"><input type="checkbox" '+(set.has(id)?'checked':'')+'><div><b>'+esc(storeName(id))+'</b><span>'+esc(id)+'</span></div><button type="button" class="ico" onclick="moveStoreRow(this,-1)">↑</button><button type="button" class="ico" onclick="moveStoreRow(this,1)">↓</button></div>').join('')}
function moveStoreRow(btn,dir){const r=btn.closest('.store-row'),p=r.parentElement;if(dir<0&&r.previousElementSibling)p.insertBefore(r,r.previousElementSibling);if(dir>0&&r.nextElementSibling)p.insertBefore(r.nextElementSibling,r)}
function bookingStoresFromUI(){const rows=Array.from(document.querySelectorAll('#bookingStores .store-row')),selected=[];rows.forEach(r=>{if(r.querySelector('input').checked)selected.push(r.dataset.store)});return{selected_stores:selected,store_priority:selected}}
function applyPreset(k){const set=(pm,st,tm,wd,sa,su)=>{el('ppm').value=pm;el('pst').value=st;el('ptm').value=tm;rT('wd',wd);rT('sa',sa);rT('su',su)};if(k==='weekday_dinner')set('weekday_first','closest','1930',[{start:'1900',end:'2030'}],[],[]);else if(k==='weekend_lunch')set('weekend_first','earliest','1130',[],[{start:'1030',end:'1300'}],[{start:'1030',end:'1300'}]);else if(k==='weekend_dinner')set('weekend_first','closest','1930',[],[{start:'1830',end:'2030'}],[{start:'1830',end:'2030'}]);else if(k==='any_available')set('date','earliest','1930',[{start:'1000',end:'2200'}],[{start:'1000',end:'2200'}],[{start:'1000',end:'2200'}]);toast('已套用策略模板，请点击保存偏好')}
function rT(k,rs){const c=el(k);c.innerHTML='';(rs||[]).forEach(r=>{const d=document.createElement('div');d.className='tr';d.innerHTML='<input type="text" value="'+escA(r.start||'')+'" placeholder="1930"><span class="sp">至</span><input type="text" value="'+escA(r.end||'')+'" placeholder="2030"><span class="x" onclick="this.parentElement.remove()">×</span>';c.appendChild(d)});if(!rs||!rs.length)c.innerHTML='<span class="mu">不预约</span>'}
function aT(k){const c=el(k);if(c.querySelector('.mu'))c.innerHTML='';const d=document.createElement('div');d.className='tr';d.innerHTML='<input type="text" placeholder="1930"><span class="sp">至</span><input type="text" placeholder="2030"><span class="x" onclick="this.parentElement.remove()">×</span>';c.appendChild(d)}
function gT(k){const ip=document.querySelectorAll('#'+k+' input'),r=[];for(let i=0;i<ip.length;i+=2){const s=ip[i].value.trim(),e=ip[i+1]?ip[i+1].value.trim():'';if(s||e)r.push({start:s,end:e})}return r}
function prefsPayload(){const st=bookingStoresFromUI();return{ui_mode:currentUIMode(),adult:+el('pa').value||2,child:+el('pc').value||0,table_type:el('pt').value||'T',phone_number:(el('pphone')?.value||'').trim(),selected_stores:st.selected_stores,store_priority:st.store_priority,day_priority_mode:el('ppm').value||'date',day_priority:pr.day_priority||['saturday','sunday','weekday'],slot_strategy:el('pst').value||'earliest',target_time:el('ptm').value.trim()||'1930',weekday_slots:gT('wd'),saturday_slots:gT('sa'),sunday_slots:gT('su')}}
async function savePrefsPayload(b,quiet){const modeSeq=uiModeSeq;try{const d=await(await fetch('/api/preferences',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)})).json();if(d.error){if(!quiet)toast(d.error);return false}pr=d.preferences||b;prefsLoaded=true;const serverMode=pr.ui_mode==='advanced'?'advanced':'simple';if(modeSeq===uiModeSeq||serverMode===currentUIMode())cacheUIMode(serverMode);else pr={...pr,ui_mode:currentUIMode()};fF(pr);dP(pr);renderBookingStores();uD();applyUIMode();if(!quiet)toast('已保存');return true}catch(e){if(!quiet)toast('保存失败');return false}}
async function sP(){const b=prefsPayload();if(stores.length&&!b.selected_stores.length){toast('请至少选择一家预约/取号门店');return false}return savePrefsPayload(b,false)}
async function saveCalendarStoresAsPrefs(){if(!selStores.length){toast('请先选择门店');return}await lP();const b={...pr,selected_stores:selStores.slice(),store_priority:selStores.slice()};if(await savePrefsPayload(b,true))toast('已保存为预约/取号门店优先级')}
function renderCloudAuth(d){cloudAuth=d||{};const st=el('cloudState');if(el('cloudUrl')&&!el('cloudUrl').value)el('cloudUrl').value=cloudAuth.base_url||'';const cfg=!!cloudAuth.configured,conn=!!cloudAuth.connected,user=cloudAuth.user_login||'',baseOK=!!cloudAuth.baseline_connected,baseCount=(cloudAuth.baseline_rollup_count||0)+(cloudAuth.baseline_latest_count||0),baseText=baseOK?(baseCount?('已验证 '+baseCount+' 条'):'已响应，暂无样本'):(conn?'未验证':'待登录'),msg=cloudAuth.last_error?('<br><span class="bad">'+esc(cloudAuth.last_error)+'</span>'):'',who=conn?(esc(user||'GitHub')+(cloudAuth.expires_at?(' · 到期 '+esc(shortTime(cloudAuth.expires_at))):'')):(cfg?'待登录 GitHub':'未连接');if(st)st.innerHTML=chip('线上基准',cfg?'服务已配置':'未连接',cfg?'ok':'warn')+chip('GitHub',who,conn?'ok':cfg?'warn':'warn')+chip('线上数据库',baseText,baseOK?'ok':conn?'warn':'warn')+chip('本机保存','只保存应用会话，不保存数据库密钥',conn?'ok':'warn')+msg+'<div class="mu mt8">'+esc(cloudAuth.provider_message||'登录只用于线上基准，不影响寿司郎认证和本机取号。')+'</div>';renderSettingsStatus()}
async function loadCloudAuth(verify){try{renderCloudAuth(await safeFetch('/api/cloud/auth'+(verify?'?verify=1':''),null,12000))}catch(e){const st=el('cloudState');if(st)st.innerHTML='<span class="bad">加载云端状态失败：'+esc(String(e.message||e))+'</span>'}}
async function saveCloudAuth(){const base=(el('cloudUrl')?.value||'').trim();try{const d=await safeFetch('/api/cloud/auth',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({base_url:base})});renderCloudAuth(d);toast(base?'已保存云端服务地址':'已清空云端服务地址');return true}catch(e){toast('保存云端设置失败：'+String(e.message||e));return false}}
async function startCloudLogin(){const base=(el('cloudUrl')?.value||'').trim();if(base&&base!==(cloudAuth.base_url||'')){if(!await saveCloudAuth())return}if(!(cloudAuth.configured||base)){toast('还没有配置云端服务地址；自建服务请在「GitHub 登录与线上基准」高级折叠里填写。');return}location.href='/api/cloud/auth/start'}
async function testCloudAuth(){try{renderCloudAuth(await safeFetch('/api/cloud/auth/test',{method:'POST'},15000));toast('GitHub 与线上数据库连接正常')}catch(e){await loadCloudAuth(true);toast('云端连接失败：'+String(e.message||e))}}
async function logoutCloudAuth(){if(!await confirmDialog('退出云端 GitHub 会话？\\n只会清空本机保存的云端 session，不影响寿司郎凭证和本机数据。'))return;try{renderCloudAuth(await safeFetch('/api/cloud/auth/logout',{method:'POST'}));toast('已退出云端')}catch(e){toast('退出云端失败：'+String(e.message||e))}}
async function lS(){await lP();await ensureStores();renderBookingStores();try{const c=await(await fetch('/api/config')).json();el('nf').value=c.feishu?.webhook||'';el('ntt').value=c.telegram?.token||'';el('ntc').value=c.telegram?.chat_id||'';el('nbu').value=c.bark?.url||'';el('nbk').value=c.bark?.key||'';el('ns').value=c.server_chan?.key||'';notifyChannels=[];if(c.feishu?.webhook)notifyChannels.push('飞书');if(c.telegram?.token&&c.telegram?.chat_id)notifyChannels.push('Telegram');if(c.bark?.url&&c.bark?.key)notifyChannels.push('Bark');if(c.server_chan?.key)notifyChannels.push('Server酱');nfc=notifyChannels.length>0;renderSettingsStatus()}catch(e){}const verifyCloud=cloudVerifyOnLoad;cloudVerifyOnLoad=false;await loadCloudAuth(verifyCloud);await loadMobileAuth();lD()}
async function sN(quiet){const b={feishu:{webhook:el('nf').value.trim()},telegram:{token:el('ntt').value.trim(),chat_id:el('ntc').value.trim()},bark:{url:el('nbu').value.trim(),key:el('nbk').value.trim()},server_chan:{key:el('ns').value.trim()}};try{const d=await(await fetch('/api/config',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)})).json();if(d.error){if(!quiet)toast(d.error);return false}notifyChannels=[];if(b.feishu.webhook)notifyChannels.push('飞书');if(b.telegram.token&&b.telegram.chat_id)notifyChannels.push('Telegram');if(b.bark.url&&b.bark.key)notifyChannels.push('Bark');if(b.server_chan.key)notifyChannels.push('Server酱');nfc=notifyChannels.length>0;renderSettingsStatus();if(!quiet){toast('已保存');loadStatus().then(()=>{if(pr&&pr.adult!==undefined)dP(pr)})}return true}catch(e){if(!quiet)toast('保存失败');return false}}
async function tN(ch){if(!await sN(true))return;try{const r=await fetch('/api/notifications/test',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({channel:ch||'all'})}),d=await r.json();if(d.error){toast(d.error);return}const bad=(d.results||[]).filter(x=>!x.ok).map(x=>x.channel+': '+x.error);toast(bad.length?'已先保存当前表单，部分发送失败：\n'+bad.join('\n'):'已先保存当前表单，测试通知已发送')}catch(e){toast('发送失败')}}
function mobileUaTime(t){try{return t?new Date(t).toLocaleString('zh-CN',{hour12:false}):'-'}catch(e){return t||'-'}}
function capLine(c){if(!c)return'<span class="bad">尚未开始</span>';const rows=[['X-App-Code',c.x_app_code],['查询凭证',c.query_auth],['User-Agent',c.user_agent],['Referer',c.referer],['预约凭证',c.reservation_auth],['微信ID',c.wechat_id],['手机号',c.phone_number],['门店',c.store_ids]];return rows.map(x=>'<span class="'+(x[1]?'ok':'bad')+'">'+esc(x[1]?'✓ ':'⏳ ')+esc(x[0])+'</span>').join(' · ')+'<br><b>完整状态：</b>'+(c.complete?'<span class="ok">已完整</span>':'<span class="bad">未完整</span>')}
function renderMobileAuth(d){const st=el('mobileAuthState');if(!st)return;const active=!!d.active,cap=d.capture||null,logs=d.logs||[];let html='<b>'+esc(active?'手机捕获中（请从「拿通行证（向导）」继续或停止）':(d.saved?'已保存':'未运行'))+'</b><br>'+esc(d.message||'')+(active?'<br>失效时间：'+esc(mobileUaTime(d.expires)):'')+'<br>CA：<code>'+esc(d.ca_path||'')+'</code><br>'+capLine(cap);if(logs.length)html+='<br><b>最近日志</b><br>'+logs.slice(-6).map(l=>esc((l.time||'')+' '+(l.message||''))).join('<br>');st.innerHTML=html}
async function loadMobileAuth(){try{renderMobileAuth(await safeFetch('/api/mobile-auth'))}catch(e){const st=el('mobileAuthState');if(st)st.innerHTML='<span class="bad">加载手机凭证状态失败：'+esc(String(e.message||e))+'</span>'}}
function chip(t,s,c){return'<div class="ci '+c+'">'+esc(t)+'：'+esc(s)+'</div>'}
function diagnosticAdvice(d){
 const cfg=d.config||{},cert=d.certificate||{},pm=d.proxy_marker||{},chain=d.proxy_chain||{},net=d.network||{},eng=d.engine||{},isWin=(d.platform||{}).goos==='windows';
 const certUntrusted=isWin?(cert.cert_exists&&(!cert.current_user_trusted||!cert.local_machine_trusted)):(cert.cert_exists&&!cert.trusted);
 if(pm.stale)return{level:'bad',title:'先修复代理残留',body:'系统代理里还有上次留下的寿司郎代理。先修复代理，再重新获取通行证或启动任务。',buttons:[{l:'修复代理',f:'repairP()'},{l:'复制诊断',f:'copyDiag()'}]};
 if(!cfg.complete)return{level:'bad',title:'先拿通行证',body:'抢预约、远程取号和读取我的单据需要完整通行证。看排队仍然可以直接用。',buttons:[{l:'拿通行证',f:'startAuth()'},{l:'先看排队',f:"go('qt')"}]};
 if(certUntrusted)return{level:'bad',title:'先信任证书',body:'证书未被系统完整信任，寿司郎小程序请求可能抓不到。按向导重新获取通行证并允许安装证书。',buttons:[{l:'重新拿通行证',f:'resetAuthAndStart()'},{l:'复制诊断',f:'copyDiag()'}]};
 if(chain.checked&&!chain.ok)return{level:'bad',title:'先处理代理链路',body:'本机代理链路自检失败。保留本页诊断信息，再修复代理或发给开发者排查。',buttons:[{l:'修复代理',f:'repairP()'},{l:'复制诊断',f:'copyDiag()'}]};
 if(net.reachable===false)return{level:'warn',title:'先确认网络',body:'当前访问寿司郎接口失败，可能是网络、地区或临时接口波动。确认网络后刷新诊断。',buttons:[{l:'刷新诊断',f:'lD()'},{l:'复制诊断',f:'copyDiag()'}]};
 if(!cfg.store_count)return{level:'warn',title:'先选常用门店',body:'选好门店后，排队、预测、可约日历和自动抢预约都会自动带入，体验会顺很多。',buttons:[{l:'选门店',f:'openGuestStorePicker()'},{l:'改预约/取号偏好',f:'openSnPrefs()'}]};
 if(!(cfg.notification_channels||[]).length)return{level:'warn',title:'建议配置通知',body:'不配置通知也能使用，但叫号提醒和抢到预约不会主动推送。',buttons:[{l:'去配置通知',f:'focusNotifySettings()'},{l:'暂时不用',f:"go('da')"}]};
 if(eng.status==='error')return{level:'bad',title:'先看运行错误',body:explainMsg(eng.message||'运行遇到问题。处理红色项后再重新启动任务。'),buttons:[{l:'查看日志',f:"openSettingsFold('fold-lo')"},{l:'复制诊断',f:'copyDiag()'}]};
 return{level:'ok',title:'本机状态正常',body:'通行证、代理、网络和通知都没有明显阻塞项。可以回首页继续查排队、预约或自动抢。',buttons:[{l:'回首页',f:"go('da')"},{l:'查可约时段',f:"enterAdvanced('ca')"}]}
}
function renderDiagnosticNext(d){
 const box=el('diagNext');if(!box)return;
 const a=diagnosticAdvice(d),buttons=journeyButtonsHTML(a.buttons);
 box.className='diag-next '+a.level;
 box.innerHTML='<h3>'+esc(a.title)+'</h3><p>'+esc(a.body)+'</p>'+(buttons?'<div class="fl g8 fw mt8">'+buttons+'</div>':'');
}
function diagDetail(d){const cfg=d.config||{},cert=d.certificate||{},pm=d.proxy_marker||{},sp=d.system_proxy||{},chain=d.proxy_chain||{},net=d.network||{},logs=(d.engine_log_tail||[]).concat((d.log_tail||[]).map(x=>({time:'',message:x}))),ports=d.ports||[],isWin=(d.platform||{}).goos==='windows';const badPorts=ports.filter(p=>!p.available&&!p.current&&!p.fallback_port).map(p=>p.name+': '+(p.error||'占用')),portNotes=ports.filter(p=>p.note).map(p=>p.name+': '+p.note),chainLines=(chain.probes||[]).map(p=>p.name+': '+(p.ok?'正常':p.skipped?'跳过':'异常')+(p.detail?'（'+p.detail+'）':''));let html='<b>下一步建议</b><br>';if(!cfg.complete)html+='先重新获取凭证参数。<br>';if(isWin&&cert.cert_exists&&!cert.current_user_trusted&&!cert.local_machine_trusted)html+='证书已生成但未信任，请重新获取凭证并允许管理员权限安装证书。<br>';if(isWin&&cert.current_user_trusted&&!cert.local_machine_trusted)html+='Windows 机器级证书未信任，PC 微信可能拒绝访问；请重新获取凭证并允许管理员权限。<br>';if(isWin&&!cert.current_user_trusted&&cert.local_machine_trusted)html+='Windows 当前用户证书未信任，请重新获取凭证补齐证书信任。<br>';if(!isWin&&cert.cert_exists&&!cert.trusted)html+='证书已生成但未信任，请重新获取凭证触发安装。<br>';if(chain.checked&&!chain.ok)html+='代理链路自检失败，请保留本页信息发给开发者。<br>';if(pm.stale)html+='发现代理残留，请先点“修复代理”。<br>';if(!net.reachable)html+='寿司郎网络不可达，先确认网络或稍后重试。<br>';html+='<br><b>证书</b>：<code>'+esc(cert.cert_path||'-')+'</code>'+(cert.trust_error?'<br>'+esc(cert.trust_error):'')+(isWin&&(cert.current_user_trusted||cert.local_machine_trusted)?'<br>CurrentUser='+esc(String(!!cert.current_user_trusted))+'；LocalMachine='+esc(String(!!cert.local_machine_trusted))+'；Disallowed='+esc(String(!!cert.disallowed)):'');if(badPorts.length||portNotes.length)html+='<br><b>端口</b>：'+esc(badPorts.concat(portNotes).join('；'));if((sp.summary||[]).length)html+='<br><b>系统代理</b>：'+esc(sp.summary.join('；'));html+='<br><b>代理链路</b>：'+esc(chain.summary||'未检查')+(chainLines.length?'<br>'+esc(chainLines.join('；')):'');if(logs.length)html+='<br><b>最近日志</b><br>'+logs.slice(-8).map(l=>esc((l.time||'')+' '+(l.message||''))).join('<br>');return html}
async function lD(){
 const box=el('dg'),detail=el('ddetail'),next=el('diagNext');if(!box)return;
 box.innerHTML='<div class="ci">诊断中…</div>';
 if(next){next.className='diag-next warn';next.innerHTML='<h3>先处理这件事</h3><p>正在检查通行证、代理、证书、网络和通知。</p>'}
 if(detail)detail.classList.add('hid');
 try{
  const d=await safeFetch('/api/diagnostics',null,20000);lastDiag=d;renderDiagnosticNext(d);
  const cfg=d.config||{},cert=d.certificate||{},pm=d.proxy_marker||{},sp=d.system_proxy||{},chain=d.proxy_chain||{},eng=d.engine||{},net=d.network||{},dp=d.ports||[],isWin=(d.platform||{}).goos==='windows';
  const miss=(cfg.missing||[]).join('、'),portIssues=dp.filter(p=>p.in_use&&!p.current&&!p.fallback_port).map(p=>p.name),portNotes=dp.filter(p=>p.note).map(p=>p.note),portText=portIssues.length?portIssues.join('、'):(portNotes.length?portNotes.join('、'):'默认端口可用'),certText=isWin?(cert.local_machine_trusted?'机器级已信任':cert.current_user_trusted?'用户级已信任':(cert.cert_exists?'未信任':'未生成')):(cert.trusted?'已信任':cert.cert_exists?'未信任':'未生成'),certClass=isWin?(cert.local_machine_trusted?'ok':cert.current_user_trusted?'warn':'bad'):(cert.trusted?'ok':'bad');
  const items=[];
  items.push(chip('凭证参数',cfg.complete?'完整':(miss||'未捕获'),cfg.complete?'ok':'bad'));
  items.push(chip('门店',cfg.store_count?cfg.store_count+' 个':'未选择',cfg.store_count?'ok':'bad'));
  items.push(chip('证书',certText,certClass));
  items.push(chip('端口',portText,portIssues.length?'bad':portNotes.length?'warn':'ok'));
  items.push(chip('代理残留',pm.stale?'发现残留':pm.active?'运行中':'未发现',pm.stale?'bad':pm.active?'warn':'ok'));
  items.push(chip('系统代理',sp.available?'可读取':'不可读取',sp.available?'ok':'warn'));
  items.push(chip('代理链路',chain.checked?(chain.ok?'正常':'异常'):'未运行',chain.checked?(chain.ok?'ok':'bad'):'warn'));
  items.push(chip('网络',net.reachable?'寿司郎可达':'不可达',net.reachable?'ok':'bad'));
  items.push(chip('通知',cfg.notification_channels?.length?cfg.notification_channels.join('、'):'未配置',cfg.notification_channels?.length?'ok':'warn'));
  items.push(chip('引擎',eng.status||'idle',eng.status==='error'?'bad':(eng.status==='booking'||eng.status==='capturing'||eng.status==='sniping')?'warn':'ok'));
  box.innerHTML=items.join('');
  if(detail){detail.innerHTML=diagDetail(d);detail.classList.remove('hid')}
 }catch(e){
  if(next){next.className='diag-next bad';next.innerHTML='<h3>先处理这件事</h3><p>诊断没有跑通。先确认本机服务还在运行，再重试或复制错误信息。</p><div class="fl g8 fw mt8"><button class="bt bt-w bt-s" onclick="lD()">重试</button></div>'}
  box.innerHTML=loadErrBoxHTML(e,'lD()','诊断')
 }
}
async function copyDiag(){if(!lastDiag)await lD();if(!lastDiag){toast('暂无诊断信息');return}const text=JSON.stringify(lastDiag,null,2);try{if(navigator.clipboard&&navigator.clipboard.writeText)await navigator.clipboard.writeText(text);else{const t=document.createElement('textarea');t.value=text;t.style.position='fixed';t.style.left='-9999px';document.body.appendChild(t);t.select();document.execCommand('copy');t.remove()}toast('已复制诊断信息')}catch(e){toast('复制失败，请手动选择诊断详情')}}
function authProbeHTML(d){const rs=d.results||[],ad=d.advice||[];let html='<b>基础接口自检</b>：'+(d.ok?'通过':'失败')+(d.store_id?'<br><b>门店</b>：'+esc(d.store||d.store_id)+' <code>'+esc(d.store_id)+'</code>':'');if(rs.length)html+='<br>'+rs.map(r=>esc(r.name||'-')+'：'+(r.ok?'正常':r.skipped?'跳过':'异常')+(r.status?' HTTP '+r.status:'')+(r.latency_ms?' '+r.latency_ms+'ms':'')+(r.detail?'（'+esc(r.detail)+'）':'')).join('<br>');if(ad.length)html+='<br><b>下一步</b><br>'+ad.map(esc).join('<br>');return html}
async function testAuthProbe(){const detail=el('ddetail');if(detail){detail.classList.remove('hid');detail.innerHTML='基础接口测试中...'}try{const r=await fetch('/api/auth/probe',{method:'POST'}),d=await r.json();if(detail)detail.innerHTML=authProbeHTML(d);if(!d.ok)toast('基础接口未通过，详情已显示在诊断区')}catch(e){if(detail)detail.innerHTML='基础接口测试失败：'+esc(String(e));toast('基础接口测试失败')}}
async function repairP(){try{const d=await(await fetch('/api/repair-proxy',{method:'POST'})).json();toast(d.ok?'代理已恢复':'修复失败，请看 doctor');lD()}catch(e){toast('修复失败')}}
async function stopProcesses(){if(!await confirmDialog('将恢复代理、停止后台抢预约/本机采集，并退出当前应用窗口。之后就可以删除 exe 或安装目录。继续？'))return;try{const r=await fetch('/api/processes/stop',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({include_self:true})}),d=await r.json();toast(d.ok?'已发送停止请求，当前应用即将退出':'部分进程未停止，请稍后再试或重启电脑')}catch(e){toast('已发送停止请求，当前应用即将退出')}}
async function uninstallAll(){if(!await confirmDialog('将恢复代理、移除证书并清理本地敏感数据。继续？'))return;try{const d=await(await fetch('/api/uninstall',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({all:true,certificates:true,system_cert:true})})).json();toast(d.ok?'已清理':'部分清理失败，请看 doctor');lD()}catch(e){toast('清理失败')}}
async function killWeChat(){if(!await confirmDialog({title:'结束 PC 微信？',body:'会强制关闭所有微信进程（WeChat、WeChatAppEx 等）。注意：未发送的消息、未保存的草稿、正在传输的文件可能丢失，正在用的小程序会闪退。结束后请重新打开 PC 微信，进寿司郎小程序点一次排队或预约。',ok:'结束微信',cancel:'取消',danger:true}))return;try{const r=await fetch('/api/wechat/kill',{method:'POST',headers:{'Content-Type':'application/json','X-Sushiro-CSRF':csrfToken}});const d=await r.json();if(r.status===403){toast('CSRF 校验失败，请刷新页面后重试');return}toast(d.ok?('已结束 '+countWeChatOK(d)+' 个微信进程，请重新打开 PC 微信'):'没有检测到微信进程，或部分未结束——可手动在任务管理器关闭')}catch(e){toast('结束微信失败：'+String((e&&e.message)||e))}}
function countWeChatOK(d){try{return(d.results||[]).filter(x=>x.status==='ok').length}catch(e){return 0}}

async function lL(){try{const ls=await(await fetch('/api/engine/logs')).json(),v=el('lv');v.innerHTML=(ls||[]).map(l=>'<div class="ll '+(l.level==='error'?'er':'')+'"><span class="lt">'+esc(l.time)+'</span><span class="lm">'+esc(l.message)+'</span></div>').join('');v.scrollTop=v.scrollHeight}catch(e){}}
function aL(e){const v=el('lv');if(!v)return;const d=document.createElement('div');d.className='ll '+(e.level==='error'?'er':'');d.innerHTML='<span class="lt">'+esc(e.time)+'</span><span class="lm">'+esc(e.message)+'</span>';v.appendChild(d);const lo=el('fold-lo');if(cp==='se'&&lo&&lo.open)v.scrollTop=v.scrollHeight}
function sse(){if(cE)cE.close();const s=new EventSource('/api/events');cE=s;s.onopen=()=>{loadStatus()};s.addEventListener('engine',e=>{try{es=JSON.parse(e.data);uE();uD();if(cp==='sn'||['idle','success','error'].includes(es.status))loadSnPlan();if(es.status==='success'&&typeof lR==='function')lR();if(['idle','success','error'].includes(es.status))loadStatus()}catch(x){}});s.addEventListener('sampling',e=>{try{spState=JSON.parse(e.data);renderDashboardSamplingCard();if(cp==='se')renderSamplingState()}catch(x){}});s.addEventListener('log',e=>{try{aL(JSON.parse(e.data))}catch(x){}});s.addEventListener('calendar',e=>{try{const d=JSON.parse(e.data);if(cp==='ca'){as=[];(d.stores||[]).forEach(st=>(st.slots||[]).forEach(x=>as.push({...x,store_name:st.store_name,store_id:st.store_id})));if(as.length)rDB()}}catch(x){}});s.addEventListener('ping',()=>{});s.onerror=()=>{s.close();cE=null;setTimeout(sse,3000)}}
init();
</script>
</body></html>
`
