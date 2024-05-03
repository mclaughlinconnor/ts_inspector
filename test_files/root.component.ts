import {ChangeDetectionStrategy, Component} from '@angular/core';

@Component({
  changeDetection: ChangeDetectionStrategy.OnPush,
  selector: 'tg-root',
  templateUrl: './root.component.pug'
})

export class RootComponent {
  public public: number = 23;

  public get getter(): number {
    return 23;
  }

  constructor() { }
}
