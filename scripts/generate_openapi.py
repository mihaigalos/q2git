#!/usr/bin/env python3
"""
Generate OpenAPI specification from swag-style annotations in Go source files.
"""

import re
import yaml
from pathlib import Path


def parse_handler_annotations(file_path):
    """Parse swag annotations from a Go file."""
    endpoints = []
    
    with open(file_path, 'r') as f:
        content = f.read()
    
    # Find all function blocks with annotations
    pattern = r'((?://\s*@\w+.*\n)+)\s*func\s+(\w+)'
    matches = re.finditer(pattern, content, re.MULTILINE)
    
    for match in matches:
        annotations = match.group(1)
        func_name = match.group(2)
        
        endpoint = parse_annotations(annotations)
        if endpoint:
            endpoints.append(endpoint)
    
    return endpoints


def parse_annotations(annotations):
    """Parse individual swag annotations into endpoint data."""
    endpoint = {}
    
    lines = annotations.strip().split('\n')
    for line in lines:
        line = line.strip('/ ')
        if not line.startswith('@'):
            continue
        
        # Parse @Tag value
        if match := re.match(r'@(\w+)\s+(.*)', line):
            key, value = match.groups()
            value = value.strip()
            
            if key == 'Summary':
                endpoint['summary'] = value
            elif key == 'Description':
                endpoint['description'] = value
            elif key == 'Tags':
                endpoint['tags'] = [value]
            elif key == 'Router':
                route_match = re.match(r'(\S+)\s+\[(\w+)\]', value)
                if route_match:
                    endpoint['path'] = route_match.group(1)
                    endpoint['method'] = route_match.group(2).lower()
            elif key == 'Param':
                param_match = re.match(r'(\w+)\s+(\w+)\s+(\w+)\s+(\w+)\s+"([^"]+)"', value)
                if param_match:
                    param_name, param_in, param_type, required, description = param_match.groups()
                    if 'parameters' not in endpoint:
                        endpoint['parameters'] = []
                    endpoint['parameters'].append({
                        'name': param_name,
                        'in': param_in,
                        'required': required == 'true',
                        'description': description,
                        'schema': {'type': param_type}
                    })
            elif key == 'Success':
                status_match = re.match(r'(\d+)\s+\{(\w+)\}\s+(\S+)\s+"([^"]+)"', value)
                if status_match:
                    status, resp_type, resp_schema, description = status_match.groups()
                    if 'responses' not in endpoint:
                        endpoint['responses'] = {}
                    endpoint['responses'][status] = {
                        'description': description,
                        'content': {
                            'application/json': {
                                'schema': {'type': 'object'}
                            }
                        }
                    }
            elif key == 'Failure':
                status_match = re.match(r'(\d+)\s+\{(\w+)\}\s+(\S+)\s+"([^"]+)"', value)
                if status_match:
                    status, resp_type, resp_schema, description = status_match.groups()
                    if 'responses' not in endpoint:
                        endpoint['responses'] = {}
                    endpoint['responses'][status] = {
                        'description': description,
                        'content': {
                            'application/json': {
                                'schema': {'type': 'object'}
                            }
                        }
                    }
            elif key == 'Produce':
                endpoint['produces'] = value
    
    return endpoint if endpoint else None


def generate_openapi_spec(endpoints):
    """Generate OpenAPI 3.0 specification."""
    spec = {
        'openapi': '3.0.3',
        'info': {
            'title': 'q2git API',
            'description': 'Query execution and Git commit service for wasmCloud',
            'version': '1.0.0',
            'contact': {
                'name': 'q2git',
            }
        },
        'servers': [
            {'url': 'http://localhost:8000', 'description': 'Local development server'}
        ],
        'tags': [
            {'name': 'query', 'description': 'Query execution and commit operations'},
            {'name': 'health', 'description': 'Health and status endpoints'},
            {'name': 'info', 'description': 'Service information'},
        ],
        'paths': {}
    }
    
    for endpoint in endpoints:
        if 'path' not in endpoint or 'method' not in endpoint:
            continue
        
        path = endpoint['path']
        method = endpoint['method']
        
        if path not in spec['paths']:
            spec['paths'][path] = {}
        
        operation = {
            'summary': endpoint.get('summary', ''),
            'description': endpoint.get('description', ''),
            'tags': endpoint.get('tags', []),
            'responses': endpoint.get('responses', {})
        }
        
        if 'parameters' in endpoint:
            operation['parameters'] = endpoint['parameters']
        
        spec['paths'][path][method] = operation
    
    return spec


def main():
    src_dir = Path(__file__).parent.parent / 'src'
    output_file = Path(__file__).parent.parent / 'openapi.yaml'
    
    all_endpoints = []
    
    # Parse all Go files in src/
    for go_file in src_dir.glob('*.go'):
        endpoints = parse_handler_annotations(go_file)
        all_endpoints.extend(endpoints)
    
    # Generate OpenAPI spec
    spec = generate_openapi_spec(all_endpoints)
    
    # Write to file
    with open(output_file, 'w') as f:
        yaml.dump(spec, f, sort_keys=False, default_flow_style=False)
    
    print(f"✓ Generated OpenAPI spec: {output_file}")
    print(f"  Found {len(all_endpoints)} endpoints")


if __name__ == '__main__':
    main()
