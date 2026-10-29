"""Opt-in real native playback, using a separate queue/state directory. Requires Windows and a cached Premium login."""
from pathlib import Path
import argparse, ctypes, hashlib, json, os, queue, subprocess, threading, time
from ctypes import wintypes

parser=argparse.ArgumentParser()
parser.add_argument('--backend',type=Path,required=True)
parser.add_argument('--profile',type=Path,required=True)
parser.add_argument('--state-dir',type=Path,required=True)
parser.add_argument('--album-id',required=True)
parser.add_argument('--seconds',type=int,default=600)
parser.add_argument('--report',type=Path,required=True)
parser.add_argument('--allow-audio',action='store_true')
args=parser.parse_args()
if not args.allow_audio:parser.error('Playback is audible at 15% volume; explicitly pass --allow-audio.')
if not 60<=args.seconds<=3600:parser.error('--seconds must be between 60 and 3600')
if args.state_dir.resolve()==(args.profile/'state').resolve():parser.error('Use a separate test state directory.')
if len(args.album_id)!=22 or not args.album_id.isalnum():parser.error('Use a Spotify album ID.')
env=os.environ.copy()
for key,path in [('XDG_CONFIG_HOME',args.profile/'config'),('XDG_CACHE_HOME',args.profile/'cache'),('XDG_STATE_HOME',args.state_dir)]:env[key]=str(path.resolve())
user_queue=args.profile/'state/spotatui/pomme_queue.json'
digest=lambda:hashlib.sha256(user_queue.read_bytes()).hexdigest() if user_queue.exists() else None
original=digest()
p=subprocess.Popen([str(args.backend.resolve())],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True,encoding='utf-8',env=env,creationflags=subprocess.CREATE_NO_WINDOW)
responses=queue.Queue();counter=0
def read():
    for line in p.stdout:
        try:responses.put(json.loads(line))
        except ValueError:responses.put(None)
threading.Thread(target=read,daemon=True).start()
def call(method,params=None):
    global counter
    counter+=1;p.stdin.write(json.dumps({'id':counter,'method':method,'params':params or {}})+'\n');p.stdin.flush()
    reply=responses.get(timeout=70)
    if not reply or reply.get('id')!=counter or 'error' in reply:raise RuntimeError('Backend call failed: '+method)
    return reply['result']
class Counters(ctypes.Structure):
    _fields_=[('cb',wintypes.DWORD),('PageFaultCount',wintypes.DWORD)]+[(name,ctypes.c_size_t) for name in ['PeakWorkingSetSize','WorkingSetSize','QuotaPeakPagedPoolUsage','QuotaPagedPoolUsage','QuotaPeakNonPagedPoolUsage','QuotaNonPagedPoolUsage','PagefileUsage','PeakPagefileUsage','PrivateUsage']]
def memory():
    stats=Counters();stats.cb=ctypes.sizeof(stats)
    if not ctypes.windll.psapi.GetProcessMemoryInfo(wintypes.HANDLE(int(p._handle)),ctypes.byref(stats),stats.cb):raise RuntimeError('Memory sampling failed')
    return {'working_set_mib':round(stats.WorkingSetSize/1048576,2),'private_mib':round(stats.PrivateUsage/1048576,2)}
samples=[];report={'status':'running','requested_seconds':args.seconds,'scope':'native engine only; frontend and human listening quality excluded','volume_percent':15,'samples':samples}
def save():args.report.write_text(json.dumps(report,indent=2),encoding='utf-8')
try:
    if not call('hello')['authed']:raise RuntimeError('A cached Premium login is required')
    call('set_volume',{'volume':15});call('play',{'kind':'album','id':args.album_id})
    deadline=time.monotonic()+60
    while True:
        state=call('state')
        if state.get('err'):raise RuntimeError('Player reported an audio/service error')
        if state['playing'] and state['posMs']>=1000:break
        if time.monotonic()>deadline:raise RuntimeError('Playback did not begin advancing')
        time.sleep(.5)
    while state['repeat']!=0:call('cycle_repeat');state=call('state')
    start=time.monotonic();last_progress=start;previous=(state['queuePos'],state['posMs']);transitions=0
    next_notice=60
    while time.monotonic()-start<args.seconds:
        state=call('state');elapsed=round(time.monotonic()-start,2)
        if state.get('err'):raise RuntimeError('Player reported an audio/service error')
        current=(state['queuePos'],state['posMs'])
        if current!=previous:last_progress=time.monotonic()
        if current[0]!=previous[0]:transitions+=1
        if time.monotonic()-last_progress>30:raise RuntimeError('Playback stopped advancing for 30 seconds')
        previous=current
        samples.append({'elapsed_seconds':elapsed,**memory(),'playing':state['playing'],'queue_position':state['queuePos'],'position_ms':state['posMs']})
        if elapsed>=next_notice:
            print(json.dumps({'elapsed_seconds':elapsed,**memory()}),flush=True);next_notice+=60;save()
        time.sleep(min(5,max(0,args.seconds-(time.monotonic()-start))))
    call('pause');call('shutdown');p.stdin.close();p.wait(timeout=12)
    report.update({'status':'passed','elapsed_seconds':round(time.monotonic()-start,2),'peak_working_set_mib':max(s['working_set_mib'] for s in samples),'peak_private_mib':max(s['private_mib'] for s in samples),'first_working_set_mib':samples[0]['working_set_mib'],'last_working_set_mib':samples[-1]['working_set_mib'],'track_transitions':transitions,'user_queue_unchanged':original==digest()})
    if p.returncode!=0 or not report['user_queue_unchanged']:raise RuntimeError('Shutdown failed or user queue changed')
except Exception as error:
    report.update({'status':'failed','error':str(error),'user_queue_unchanged':original==digest()});raise
finally:
    if p.poll() is None:
        try:call('pause');call('shutdown');p.stdin.close();p.wait(timeout=8)
        except Exception:p.kill();p.wait()
    save()
print(json.dumps({key:value for key,value in report.items() if key!='samples'}))
