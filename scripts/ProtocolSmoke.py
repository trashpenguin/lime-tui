from pathlib import Path
import ctypes, json, os, queue, subprocess, sys, threading, time, tempfile

base = Path(__file__).resolve().parent
exe = Path(sys.argv[1]).resolve()
sandbox = tempfile.TemporaryDirectory(prefix='lime-protocol-')
profile = Path(sandbox.name)
profile.mkdir(exist_ok=True)
state = profile/'state'/'spotatui'/'pomme_queue.json'
state.parent.mkdir(parents=True,exist_ok=True)
tracks=[dict(id=f'{i:022}',title=f'Fixture {i}',artist='Offline test',album='Fixture',kind='songs',durMs=300000,art='') for i in range(1,4)]
state.write_text(json.dumps(dict(tracks=tracks,index=1,position=12000,volume=70,shuffle=False,repeat=0)),encoding='utf-8')
env=os.environ.copy()
for key,part in [('XDG_CONFIG_HOME','config'),('XDG_CACHE_HOME','cache'),('XDG_STATE_HOME','state')]: env[key]=str(profile/part)
flags=subprocess.CREATE_NO_WINDOW if os.name=='nt' else 0
p=subprocess.Popen([str(exe)],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True,encoding='utf-8',env=env,creationflags=flags)
responses=queue.Queue()
def read():
    for line in p.stdout: responses.put(json.loads(line))
threading.Thread(target=read,daemon=True).start()
counter=0
def call(method,params=None,error=False):
    global counter
    counter+=1
    p.stdin.write(json.dumps(dict(id=counter,method=method,params=params or {}))+'\n');p.stdin.flush()
    reply=responses.get(timeout=12)
    assert reply['id']==counter,reply
    assert ('error' in reply)==error,reply
    return reply.get('result',reply.get('error'))

try:
    assert call('hello')=={'protocol':1,'authed':False}
    s=call('state');assert len(s['queue'])==3 and s['queuePos']==1 and not s['playing'] and s['posMs']==12000
    call('queue_edit',dict(action='move',index=1,target=2))
    s=call('state');assert s['queuePos']==2 and s['now']['id']==tracks[1]['id']
    call('queue_edit',dict(action='remove',index=0))
    s=call('state');assert s['queuePos']==1 and s['now']['id']==tracks[1]['id']
    call('set_volume',dict(volume=42));assert call('state')['volume']==42
    call('seek',dict(atMs=900000));assert call('state')['posMs']==300000
    call('toggle_shuffle');assert call('state')['shuffle']
    for repeat in [1,2,0]:call('cycle_repeat');assert call('state')['repeat']==repeat
    call('play',dict(kind='songs',id='invalid'),error=True)
    call('search',dict(query=''),error=True)
    call('queue_edit',dict(action='remove',index=999),error=True)
    call('unknown_fixture',error=True)
    assert call('spectrum') == [0.0]*32
    call('queue_edit',dict(action='clear'))
    s=call('state');assert s['queue']==[] and s['now'] is None and not s['playing']
    idle_memory=None
    if os.name=='nt':
        from ctypes import wintypes
        class Counters(ctypes.Structure):
            _fields_=[('cb',wintypes.DWORD),('PageFaultCount',wintypes.DWORD)]+[(n,ctypes.c_size_t) for n in ['PeakWorkingSetSize','WorkingSetSize','QuotaPeakPagedPoolUsage','QuotaPagedPoolUsage','QuotaPeakNonPagedPoolUsage','QuotaNonPagedPoolUsage','PagefileUsage','PeakPagefileUsage','PrivateUsage']]
        c=Counters();c.cb=ctypes.sizeof(c)
        if ctypes.windll.psapi.GetProcessMemoryInfo(wintypes.HANDLE(int(p._handle)),ctypes.byref(c),c.cb):idle_memory=round(c.WorkingSetSize/1048576,2)
    call('shutdown')
    p.wait(timeout=8)
    assert p.returncode==0,p.stderr.read()
    saved=json.loads(state.read_text());assert saved['volume']==42 and saved['tracks']==[]
finally:
    if p.poll() is None:p.kill();p.wait()

# A malformed/truncated producer cannot cause unbounded line allocation.
oversized=subprocess.run([str(exe)],input='x'*70000+'\n',capture_output=True,text=True,env=env,creationflags=flags,timeout=10)
assert oversized.returncode!=0 and '64 KiB' in oversized.stderr
# EOF is ownership loss and must terminate the helper even without shutdown.
eof=subprocess.run([str(exe)],input='',capture_output=True,text=True,env=env,creationflags=flags,timeout=10)
assert eof.returncode==0,eof.stderr
result={'rpc_requests':counter,'status':'passed','oversized_input':'rejected','owner_eof':'exited','queue_persistence':'passed','backend_idle_unauthenticated_working_set_mib':idle_memory,'live_spotify_playback':'not tested'}
(base/'protocol-smoke-result.json').write_text(json.dumps(result,indent=2))
print(json.dumps(result))
