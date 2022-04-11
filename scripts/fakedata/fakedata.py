# -*- coding: utf-8 -*-

import os
import urllib
import logging
import requests
import yaml


logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s]\t%(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)

qs = urllib.parse.urlencode({
    "page": 1,
    "size": 10000,
})


class Store:
    def __init__(self):
        self.users = {}
        self.clusters = {}
        self.tenants = {}


class DataLoader:

    def __init__(self):
        self._load()

    def _load(self):
        with open("datas.yaml", "r") as f:
            datas = yaml.safe_load(f)
        self.clusters = datas['clusters']
        self.users = datas['users']
        self.tenants = datas['tenants']
        self.default_limit_range = datas['defaultLimitRange']


class GemsData:

    def __init__(self, host, username, password):
        self.store = Store()
        self.url = host
        self.username = username
        self.password = password

        self.jwt_token = self.get_jwt_token()
        self.data_loader = DataLoader()
        self.load_exist_datas()

    def post(self, path, data):
        url = self.url + path
        resp = requests.post(url, json=data, headers={'Authorization': 'Bearer ' + self.jwt_token})
        if resp.status_code not in [200, 201]:
            raise Exception('post %s failed, code is %d' % (path, resp.status_code))
        datas = resp.json()
        return datas['Data']

    def list(self, path):
        url = self.url + path
        resp = requests.get(url, headers={'Authorization': 'Bearer ' + self.jwt_token})
        if resp.status_code not in [200, 201]:
            raise Exception('get %s failed, code is %d' % (path, resp.status_code))
        datas = resp.json()
        return datas['Data']['List']

    def load_exist_datas(self):
        cluster_list = self.list('/api/v1/cluster?' + qs)
        user_list = self.list('/api/v1/user?' + qs)
        tenant_list = self.list('/api/v1/tenant?' + qs)

        for cluster in cluster_list:
            self.store.clusters[cluster['ClusterName']] = cluster
        for user in user_list:
            self.store.users[user['Username']] = user
        for tenant in tenant_list:
            self.store.tenants[tenant['TenantName']] = tenant

    def get_jwt_token(self):
        url = self.url + '/api/v1/login'
        data = {
            'source': 'account',
            'username': self.username,
            'password': self.password
        }
        response = requests.post(url, json=data)
        if response.status_code == 200:
            datas = response.json()
            return datas['Data']['token']
        else:
            raise Exception('get jwt token failed')

    def create_clusters(self):
        for cluster in self.data_loader.clusters:
            cluster_name = cluster['name']
            if cluster_name in self.store.clusters:
                logging.info("cluster %s already exists" % cluster_name)
                continue
            fname = os.path.join("kubeconfigs", cluster_name + ".yaml")
            if not os.path.exists(fname):
                raise Exception('kubeconfig file for cluster %s not found' % cluster_name)
            with open(fname, "r") as f:
                kubeconfig = yaml.safe_load(f)
            url = self.url + '/api/v1/cluster'
            data = {
                'ClusterName': cluster_name,
                'KubeConfig': kubeconfig,
                'Vendor': 'selfhosted',
                'ImageRepo': 'docker.io/kubegems',
                'DefaultStorageClass': 'standard',
            }
            cluster_data = self.post('/api/v1/cluster', data)
            self.store.clusters[cluster_name] = cluster_data

    def create_users(self):
        for user in self.data_loader.users:
            user_name = user['name']
            if user_name in self.store.users:
                logging.info("user %s already exists" % user_name)
                continue
            user['username'] = user_name
            userdata = self.post('/api/v1/user', user)
            self.store.users[user_name] = userdata
            logging.info("user %s created" % user_name)

    def create_tenants(self):
        for tenant in self.data_loader.tenants:
            tenant_name = tenant['name']
            if tenant_name in self.store.tenants:
                logging.info("tenant %s already exists" % tenant_name)
                continue
            data = {
                'TenantName': tenant_name,
                'Remark': tenant_name,
            }
            tenant_data = self.post('/api/v1/tenant', data)
            self.store.tenants[tenant_name] = tenant_data
            logging.info("tenant %s created" % tenant_name)

    def create_tenants_quotas(self):
        for tenant in self.data_loader.tenants:
            tenant_name = tenant['name']
            tenant_data = self.store.tenants[tenant_name]
            tenant_quotas = self._get_tenant_quotas(tenant_data) 
            for quota in tenant.get('quotas', []):
                cluster_name = quota['cluster']
                clusterid = self.store.clusters[cluster_name]['ID']
                if clusterid in tenant_quotas:
                    logging.info("tenant %s on cluster %s already has quota" % (tenant_name, cluster_name))
                    continue
                quotadata = {
                    'ClusterID': clusterid,
                    'TenantID': tenant_data['ID'],
                    'Content': quota['content'],
                }
                self.post('/api/v1/tenant/{0}/tenantresourcequota'.format(tenant_data['ID']), quotadata)

    def _get_tenant_quotas(self, tenant):
        quotas = self.list('/api/v1/tenant/{0}/tenantresourcequota?{1}'.format(tenant['ID'], qs))
        return {quota['ClusterID']: quota for quota in quotas}

    def add_tenant_members(self):
        for tenant in self.data_loader.tenants:
            tenant_name = tenant['name']
            tenantdata = self.store.tenants[tenant_name]
            exists_members = self._get_tenant_members(tenantdata)
            for role, members in tenant['members'].items():
                for member in members:
                    uid = self.store.users[member]['ID']
                    if uid in exists_members:
                        logging.info("user %s already in tenant %s" % (member, tenant_name))
                        continue
                    tenantid = tenantdata['ID']
                    data = {
                        "TenantID": tenantid,
                        "UserID": uid,
                        "Role": role,
                    }
                    self.post('/api/v1/tenant/{0}/user'.format(tenantid), data)
                    logging.info("user %s added to tenant %s" % (member, tenant_name))

    def _get_tenant_members(self, tenant):
        member_list = self.list('/api/v1/tenant/{0}/user?{1}'.format(tenant['ID'], qs))
        return { member['ID']: member for member in member_list }

    def create_projects(self):
        for  tenant in self.data_loader.tenants:
            tenant_name = tenant['name']
            exists_projects = self._get_tenant_projects(self.store.tenants[tenant_name])
            if 'projects' not in self.store.tenants[tenant_name]:
                self.store.tenants[tenant_name]['projects'] = exists_projects

            for project in tenant['projects']:
                project_name = project['name']
                if project_name in exists_projects:
                    logging.info("project %s already exists" % project_name)
                    continue
                data = {
                    "ProjectName": project_name,
                    "ProjectAlias": project_name,
                    "Remark": project_name,
                }
                project_data = self.post('/api/v1/tenant/{0}/project'.format(self.store.tenants[tenant['name']]['ID']), data)
                if 'projects' in self.store.tenants[tenant_name]:
                    self.store.tenants[tenant_name]['projects'][project_name] = project_data
                else:
                    self.store.tenants[tenant_name]['projects'] = {project_name: project_data}
                logging.info("project %s created" % project_name)

    def _get_tenant_projects(self, tenant):
        project_list = requests.get(self.url + '/api/v1/tenant/' + str(tenant['ID']) + '/project?' + qs, headers={'Authorization': 'Bearer ' + self.jwt_token}).json()
        projects = {
            project['ProjectName']: project
            for project in project_list['Data']['List']
        }
        return projects

    def add_project_members(self):
        for  tenant in self.data_loader.tenants:
            tenant_name = tenant['name']
            for project in tenant['projects']:
                project_name = project['name']
                project_data = self.store.tenants[tenant_name]['projects'][project_name]
                exists_members = self._get_project_members(project_data)
                for role, members in project['members'].items():
                    for member in members:
                        uid = self.store.users[member]['ID']
                        if uid in exists_members:
                            logging.info("user %s already in project %s" % (member, project_name))
                            continue
                        project_id = project_data['ID']
                        data = {
                            "ProjectID": project_id,
                            "UserID": uid,
                            "Role": role,
                        }
                        self.post('/api/v1/project/{0}/user'.format(project_id), data)
                        logging.info("user %s added to project %s" % (member, project_name))

    def _get_project_members(self, project):
        member_list = self.list('/api/v1/project/{0}/user?{1}'.format(project['ID'], qs))
        return { member['ID']: member for member in member_list }

    def create_environments(self):
        for tenant in self.data_loader.tenants:
            tenant_name = tenant['name']
            for project in tenant['projects']:
                project_name = project['name']
                project_data = self.store.tenants[tenant_name]['projects'][project_name]
                envs = self._get_environments(project_data)
                for environment in project['environments']:
                    environment_name = environment['name']
                    environment_cluster = environment['cluster']
                    environment_namespace = environment['namespace']
                    if environment_name in envs:
                        if 'environments' not in self.store.tenants[tenant_name]['projects'][project_name]:
                            self.store.tenants[tenant_name]['projects'][project_name]['environments'] = { environment_name: envs[environment_name] }
                        else:
                            self.store.tenants[tenant_name]['projects'][project_name]['environments'][environment_name] = envs[environment_name]
                        logging.info("environment %s already exists" % environment_name)
                        continue
                    postdata = {
                        'EnvironmentName': environment_name,
                        'Namespace': environment_namespace,
                        'MetaType': 'dev',
                        'TenantID': self.store.tenants[tenant_name]['ID'],
                        'ClusterID': self.store.clusters[environment_cluster]['ID'],
                        'ProjectID': project_data['ID'],
                        'ResourceQuota': environment['quota'],
                        'LimitRange': self.data_loader.default_limit_range
                    }
                    environment_data = self.post('/api/v1/project/{0}/environment'.format(project_data['ID']), postdata)
                    if 'environments' in self.store.tenants[tenant_name]['projects'][project_name]:
                        self.store.tenants[tenant_name]['projects'][project_name]['environments'][environment_name] = environment_data
                    else:
                        self.store.tenants[tenant_name]['projects'][project_name]['environments'] = {environment_name: environment_data}
                    logging.info("environment %s created" % environment_name)

    def _get_environments(self, project_data):
        environment_list = self.list('/api/v1/project/{0}/environment?{1}'.format(project_data['ID'], qs))
        return { env['EnvironmentName']:env for env in environment_list }

    def add_environment_members(self):
        for tenant in self.data_loader.tenants:
            tenant_name = tenant['name']
            for project in tenant['projects']:
                project_name = project['name']
                for environment in project['environments']:
                    environment_name = environment['name']
                    environment_data = self.store.tenants[tenant_name]['projects'][project_name]['environments'][environment_name]
                    environment_id = environment_data['ID']
                    exists_members = self._get_environment_members(environment_data)
                    for role, members in environment.get('members', {}).items():
                        for member  in members:
                            uid = self.store.users[member]['ID']
                            if uid in exists_members:
                                logging.info("user %s already in environment %s" % (member, environment_name))
                                continue
                            uid = self.store.users[member]['ID']
                            data = {
                                "EnvironmentID": environment_id,
                                "UserID": uid,
                                "Role": role,
                            }
                            self.post('/api/v1/environment/{0}/user'.format(environment_data['ID']), data)
                            logging.info("user %s added to environment %s" % (member, environment_name))
    
    def _get_environment_members(self, environment):
        member_list = self.list('/api/v1/environment/{0}/user?{1}'.format(environment['ID'], qs))
        return { member['ID']: member for member in member_list }

    def create_project_scope_applications(self):
        for tenant in self.data_loader.tenants:
            tenant_name = tenant['name']
            tenant_id = self.store.tenants[tenant_name]['ID']
            for project in tenant['projects']:
                project_name = project['name']
                project_id = self.store.tenants[tenant_name]['projects'][project_name]['ID']
                for app in project.get('applications', []):
                    content = self._get_app_manifests(app['name'])
                    url = self.url + '/api/v1/tenant/{tenant_id}/project/{project_id}/manifests/{name}/files'.format(
                        tenant_id=tenant_id,
                        project_id=project_id,
                        name=app['name'],
                    ) 
                    resp = requests.put(url, json=content, headers={'Authorization': 'Bearer ' + self.jwt_token})
                    if resp.status_code not in [200, 201]:
                        raise Exception("Failed to create application manifests %s, code is %d, %s" % (app['name'], resp.status_code, resp.text))

    def create_environment_scope_applications(self):
        '/api/v1/tenant/1/project/1/environment/14/applications'
        pass

    def _get_app_manifests(self, name):
        app_path = os.path.join("applications", name)
        files = os.listdir(app_path)
        ret = []
        for fname in files:
            with open(os.path.join(app_path, fname), "r") as f:
                ret.append({
                    'name': fname,
                    'content': f.read()
                })
        return ret


if __name__ == '__main__':
    d = GemsData("http://localhost:8045", "admin", "demo!@#admin")
    d.create_clusters()
    d.create_users()

    d.create_tenants()
    d.create_tenants_quotas()
    d.add_tenant_members()

    d.create_projects()
    d.add_project_members()

    d.create_environments()
    d.add_environment_members()

    d.create_project_scope_applications()