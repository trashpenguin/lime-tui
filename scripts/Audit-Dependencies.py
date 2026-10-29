"""Audit the exact Windows runtime/build graph; preserve findings and bound assessments to source hashes."""
import argparse, hashlib, json, re, subprocess, urllib.request
from pathlib import Path

parser=argparse.ArgumentParser()
parser.add_argument('--manifest-path',type=Path,default=Path(__file__).resolve().parents[1]/'backend/Cargo.toml')
parser.add_argument('--assessments',type=Path,default=Path(__file__).resolve().parents[1]/'DEPENDENCY-ASSESSMENTS.json')
parser.add_argument('--out',type=Path,default=Path('dependency-audit.json'))
args=parser.parse_args()
backend=args.manifest_path.resolve().parent
tree=subprocess.check_output(['cargo','tree','--locked','--manifest-path',str(args.manifest_path),'--no-default-features','--features','pomme-bridge','--target','x86_64-pc-windows-msvc','--prefix','none','--edges','normal,build','--format','{p}'],text=True,encoding='utf-8')
packages=sorted(set(re.findall(r'^([\w-]+) v([^\s]+)',tree,re.M)))
queries=[{'package':{'name':name,'ecosystem':'crates.io'},'version':version} for name,version in packages if name!='spotatui']
def request(url,data=None):
    req=urllib.request.Request(url,data=None if data is None else json.dumps(data).encode(),headers={'Content-Type':'application/json','User-Agent':'Lime-release-audit/1'})
    with urllib.request.urlopen(req,timeout=90) as response:return json.load(response)
results=request('https://api.osv.dev/v1/querybatch',{'queries':queries})['results']
assessments=json.loads(args.assessments.read_text(encoding='utf-8'))
lock_hash=hashlib.sha256((backend/'Cargo.lock').read_bytes()).hexdigest()
def tree_digest(path):
    digest=hashlib.sha256()
    for file in sorted(path.rglob('*.rs')):
        digest.update(file.relative_to(path).as_posix().encode()+b'\0'+hashlib.sha256(file.read_bytes()).digest())
    return digest.hexdigest()
findings=[];details={}
for query,result in zip(queries,results):
    for match in result.get('vulns',[]):
        identifier=match['id']
        if identifier not in details:details[identifier]=request('https://api.osv.dev/v1/vulns/'+identifier)
        item={'package':query['package']['name'],'version':query['version'],'id':identifier,'status':'unassessed'}
        if details[identifier].get('withdrawn'):item['status']='withdrawn'
        for assessment in assessments['assessments']:
            if any(item[key]!=assessment[key] for key in ['package','version','id']):continue
            valid=assessments['cargo_lock_sha256']==lock_hash
            for relative,digest in assessment.get('evidence_sha256',{}).items():
                file=backend/relative
                valid=valid and file.is_file() and hashlib.sha256(file.read_bytes()).hexdigest()==digest
            for relative,digest in assessment.get('source_tree_sha256',{}).items():
                folder=backend/relative
                valid=valid and folder.is_dir() and tree_digest(folder)==digest
            if valid:
                item.update({key:assessment[key] for key in ['status','justification','impact_statement','references']})
            else:item['assessment_error']='Source or lockfile changed; review again.'
        findings.append(item)
report={'target':'x86_64-pc-windows-msvc','features':['pomme-bridge'],'cargo_lock_sha256':lock_hash,'packages_checked':len(queries),'findings':findings,'unassessed_findings':sum(f['status']=='unassessed' for f in findings),'advisories':{identifier:{key:entry[key] for key in ['id','summary','published','modified','withdrawn','aliases','affected','references','severity'] if key in entry} for identifier,entry in details.items()}}
args.out.write_text(json.dumps(report,indent=2),encoding='utf-8')
print(json.dumps({key:report[key] for key in ['packages_checked','unassessed_findings']}))
for finding in findings:print(f"{finding['package']} {finding['version']}: {finding['id']} — {finding['status']}")
raise SystemExit(1 if report['unassessed_findings'] else 0)
